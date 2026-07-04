package verify

import (
	"path/filepath"
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
		if isFigurativeTestPhrase(text, m[0], m[1]) {
			continue
		}
		add(verifiers.ClaimTestPass, text[m[0]:m[1]], "")
	}
	for _, m := range reCommitted.FindAllStringIndex(text, -1) {
		if isHistoricalOrFigurativeCommit(text, m[0], m[1]) {
			continue
		}
		add(verifiers.ClaimCommitted, text[m[0]:m[1]], "")
	}
	for _, m := range rePushed.FindAllStringIndex(text, -1) {
		if isFigurativePush(text, m[0], m[1]) {
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
		if isFigurativeCommandRan(text, m[0], m[1]) {
			continue
		}
		add(verifiers.ClaimCommandRan, text[m[0]:m[1]], "")
	}
	for _, m := range reCommandSucceeded.FindAllStringIndex(text, -1) {
		if isConditionalPhrase(text, m[0]) {
			continue
		}
		add(verifiers.ClaimCommandSucceeded, text[m[0]:m[1]], "")
	}
	for _, m := range reComplete.FindAllStringIndex(text, -1) {
		if isColloquialDone(text, m[0], m[1]) {
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
	if isFileClaim(t) && verifiers.LooksLikePath(target) {
		if strings.Contains(target, "/") || strings.Contains(target, ".") {
			return 2
		}
	}
	known := map[string]bool{"readme": true, "makefile": true, "dockerfile": true}
	base := strings.ToLower(strings.TrimSuffix(strings.ToLower(target), filepath.Ext(target)))
	if known[base] || known[strings.ToLower(filepath.Base(target))] {
		return 2
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

func stripSummarySection(text string) string {
	exec, _ := segmentProse(text)
	return exec
}

func isFigurativeTestPhrase(text string, start, end int) bool {
	snippet := strings.ToLower(text[start:end])
	if strings.Contains(snippet, "your patience") {
		return true
	}
	after := strings.ToLower(strings.TrimSpace(text[end:]))
	if strings.HasPrefix(after, "in the spec") || strings.HasPrefix(after, "in the dashboard") {
		return true
	}
	window := strings.ToLower(text[max(0, start-20):min(len(text), end+30)])
	return strings.Contains(window, "last week") || strings.Contains(window, "in ci")
}

func isHistoricalOrFigurativeCommit(text string, start, end int) bool {
	window := strings.ToLower(text[max(0, start-20):min(len(text), end+40)])
	if strings.Contains(window, "commit to ") {
		return true
	}
	for _, phrase := range []string{
		"earlier", "yesterday", "you committed", "pre-commit", "amend committed",
		"changelog says", "changelog notes", "do not amend", "hook committed",
	} {
		if strings.Contains(window, phrase) {
			return true
		}
	}
	return false
}

func isFigurativePush(text string, start, end int) bool {
	window := strings.ToLower(text[max(0, start-10):min(len(text), end+30)])
	for _, phrase := range []string{
		"pushed for", "pushed forward", "pushed to the docs", "pushed to clarify",
	} {
		if strings.Contains(window, phrase) {
			return true
		}
	}
	return false
}

func isFigurativeCommandRan(text string, start, end int) bool {
	after := strings.ToLower(strings.TrimSpace(text[end:]))
	return strings.HasPrefix(after, "in my head") || strings.HasPrefix(after, "into ") ||
		strings.HasPrefix(after, "output ")
}

func isConditionalPhrase(text string, start int) bool {
	before := strings.ToLower(text[max(0, start-20):start])
	return strings.Contains(before, "if ") || strings.Contains(before, "if the ")
}

func isColloquialDone(text string, start, end int) bool {
	snippet := strings.ToLower(strings.TrimSpace(text[start:end]))
	if snippet == "done" || snippet == "done." {
		return true
	}
	if strings.HasPrefix(snippet, "we're done") || strings.HasPrefix(snippet, "we are done") {
		return true
	}
	if strings.Contains(snippet, "done building") || strings.Contains(snippet, "done with") {
		return true
	}
	if strings.Contains(snippet, "nothing left to do on your end") {
		return true
	}
	return false
}

// HasActionProse reports whether prose contains action-oriented claims.
func HasActionProse(claims []verifiers.Claim) bool {
	return HasLocalActionProse(claims)
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
