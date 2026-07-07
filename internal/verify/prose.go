package verify

import (
	"regexp"
	"strings"

	"github.com/fristovic/snitch/internal/verify/verifiers"
)

var (
	reTestPass = regexp.MustCompile(`(?i)\b(all tests pass(?:ed)?|tests pass(?:ed)?|tests are passing|test suite pass(?:ed)?|all tests green|verified (?:with|using) tests)\b`)
	reCommitted = regexp.MustCompile(`(?i)\b(i(?:'ve| have)? committed|committed (?:the |these )?changes|commit(?:ted)? (?:to|the) git)\b`)
	rePushed = regexp.MustCompile(`(?i)\b(i(?:'ve| have)? pushed|pushed to (?:remote|origin|github))\b`)
	reFileCreated = regexp.MustCompile(`(?i)\b(created|added|wrote) (?:the )?(?:file )?(` + filePathPattern + `)`)
	reFileModified = regexp.MustCompile(`(?i)\b(updated|modified|edited|changed) (?:the )?(?:file )?(` + filePathPattern + `)`)
	reFileDeleted = regexp.MustCompile(`(?i)\b(deleted|removed) (?:the )?(?:file )?(` + filePathPattern + `)`)
	reFileDeletedBacktick = regexp.MustCompile("(?i)\\b(deleted|removed) (?:the )?(?:file )?`([^`]+)`")
	reFileModifiedBacktick = regexp.MustCompile("(?i)\\b(updated|modified|edited|changed) (?:the )?(?:file )?`([^`]+)`")
	reFileCreatedBacktick = regexp.MustCompile("(?i)\\b(created|added|wrote) (?:the )?(?:file )?`([^`]+)`")
	reCommandRan = regexp.MustCompile(`(?i)\b(?:i\s+)?(?:ran|executed)\s+(?:the\s+)?(?:\w+\s+)*command\b`)
	reCommandSucceeded = regexp.MustCompile(`(?i)\b(?:(?:command|build)\s+(?:ran|run|completed|succeeded|successful(?:ly)?)|(?:successfully|cleanly)\s+ran)\b`)
	reComplete = regexp.MustCompile(`(?i)\b(fully implemented|implementation (?:is )?complete|(?:all )?tasks? (?:are )?done|all done|nothing left to do|ready to ship)\b`)
)

const filePathPattern = "[`" + `'"']?([\w./-]+(?:\.\w+)?)[` + `'"']?`

// ExtractProseClaims finds high-confidence natural-language claims in assistant text.
func ExtractProseClaims(text string) []verifiers.Claim {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	execText, recapText := segmentProse(text)
	var claims []verifiers.Claim
	claims = append(claims, extractProseFromSegment(execText, "execution")...)
	if recapText != "" {
		claims = append(claims, extractProseFromSegment(recapText, "recap")...)
	}
	return claims
}

func extractProseFromSegment(text, segment string) []verifiers.Claim {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	var claims []verifiers.Claim
	seen := make(map[string]bool)

	add := func(t verifiers.ClaimType, quote, target string) {
		target = verifiers.NormalizePathToken(target)
		if isFileClaim(t) && !verifiers.LooksLikePath(target) {
			return
		}
		key := string(t) + "|" + target + "|" + quote + "|" + segment
		if seen[key] {
			return
		}
		seen[key] = true
		desc := quote
		if target != "" {
			desc = string(t) + " " + target
		}
		claims = append(claims, verifiers.Claim{
			Type:        t,
			Source:      "prose",
			Target:      target,
			Quote:       quote,
			Description: desc,
			Segment:     segment,
			Confidence:  scoreConfidence(t, target, quote, segment),
		})
	}

	for _, m := range reTestPass.FindAllStringIndex(text, -1) {
		if shouldSuppressClaim(verifiers.ClaimTestPass, text, m[0], m[1]) {
			continue
		}
		add(verifiers.ClaimTestPass, text[m[0]:m[1]], "")
	}
	for _, m := range reCommitted.FindAllStringIndex(text, -1) {
		if shouldSuppressClaim(verifiers.ClaimCommitted, text, m[0], m[1]) {
			continue
		}
		add(verifiers.ClaimCommitted, text[m[0]:m[1]], "")
	}
	for _, m := range rePushed.FindAllStringIndex(text, -1) {
		if shouldSuppressClaim(verifiers.ClaimPushed, text, m[0], m[1]) {
			continue
		}
		add(verifiers.ClaimPushed, text[m[0]:m[1]], "")
	}
	addBacktickFileClaims(text, reFileCreatedBacktick, verifiers.ClaimFileCreated, add)
	addBacktickFileClaims(text, reFileModifiedBacktick, verifiers.ClaimFileModified, add)
	addBacktickFileClaims(text, reFileDeletedBacktick, verifiers.ClaimFileDeleted, add)
	for _, m := range reFileCreated.FindAllStringSubmatch(text, -1) {
		if len(m) >= 3 {
			add(verifiers.ClaimFileCreated, m[0], strings.Trim(m[2], `"'`+"`"))
		}
	}
	for _, m := range reFileModified.FindAllStringSubmatch(text, -1) {
		if len(m) >= 3 {
			add(verifiers.ClaimFileModified, m[0], strings.Trim(m[2], `"'`+"`"))
		}
	}
	for _, m := range reFileDeleted.FindAllStringSubmatch(text, -1) {
		if len(m) >= 3 {
			add(verifiers.ClaimFileDeleted, m[0], strings.Trim(m[2], `"'`+"`"))
		}
	}
	for _, m := range reCommandRan.FindAllStringIndex(text, -1) {
		if shouldSuppressClaim(verifiers.ClaimCommandRan, text, m[0], m[1]) {
			continue
		}
		add(verifiers.ClaimCommandRan, text[m[0]:m[1]], "")
	}
	for _, m := range reCommandSucceeded.FindAllStringIndex(text, -1) {
		if shouldSuppressClaim(verifiers.ClaimCommandSucceeded, text, m[0], m[1]) {
			continue
		}
		add(verifiers.ClaimCommandSucceeded, text[m[0]:m[1]], "")
	}
	for _, m := range reComplete.FindAllStringIndex(text, -1) {
		if shouldSuppressClaim(verifiers.ClaimStub, text, m[0], m[1]) {
			continue
		}
		add(verifiers.ClaimStub, text[m[0]:m[1]], "")
	}
	return claims
}

// segmentProse splits assistant text into execution and recap segments.
func segmentProse(text string) (execution, recap string) {
	lower := strings.ToLower(text)
	best := -1
	for _, marker := range []string{"\n### summary", "\n## summary", "\n---\n", "\nsummary of changes", "### summary", "## summary"} {
		if i := strings.Index(lower, marker); i >= 0 {
			if best < 0 || i < best {
				best = i
			}
		}
	}
	if best < 0 {
		return text, ""
	}
	return strings.TrimSpace(text[:best]), strings.TrimSpace(text[best:])
}

func scoreConfidence(t verifiers.ClaimType, target, quote, segment string) int {
	if segment == "recap" {
		return 1
	}
	if strings.Contains(quote, "`") && target != "" {
		return 3
	}
	return 2
}

func isFileClaim(t verifiers.ClaimType) bool {
	switch t {
	case verifiers.ClaimFileCreated, verifiers.ClaimFileModified, verifiers.ClaimFileDeleted:
		return true
	default:
		return false
	}
}

func addBacktickFileClaims(text string, re *regexp.Regexp, t verifiers.ClaimType, add func(verifiers.ClaimType, string, string)) {
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		if len(m) >= 3 {
			add(t, m[0], m[2])
		}
	}
}

// HasLocalActionProse reports prose claims that imply the agent mutated state this turn.
func HasLocalActionProse(claims []verifiers.Claim) bool {
	for _, c := range claims {
		if c.Source != "prose" {
			continue
		}
		switch c.Type {
		case verifiers.ClaimCommitted, verifiers.ClaimPushed,
			verifiers.ClaimFileCreated, verifiers.ClaimFileModified, verifiers.ClaimFileDeleted,
			verifiers.ClaimCommandRan, verifiers.ClaimCommandSucceeded, verifiers.ClaimStub:
			return true
		}
	}
	return false
}
