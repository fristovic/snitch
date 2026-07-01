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
	reFileCreated = regexp.MustCompile(`(?i)\b(created|added|wrote) (?:the )?(?:file )?[` + "`" + `'"']?([\w./-]+(?:\.\w+)?)[` + "`" + `'"']?`)
	reFileModified = regexp.MustCompile(`(?i)\b(updated|modified|edited|changed) (?:the )?(?:file )?[` + "`" + `'"']?([\w./-]+(?:\.\w+)?)[` + "`" + `'"']?`)
	reFileDeleted = regexp.MustCompile(`(?i)\b(deleted|removed) (?:the )?(?:file )?[` + "`" + `'"']?([\w./-]+(?:\.\w+)?)[` + "`" + `'"']?`)
	reCommandRan = regexp.MustCompile(`(?i)\b(ran|executed) (?:the )?command\b`)
)

// ExtractProseClaims finds high-confidence natural-language claims in assistant text.
func ExtractProseClaims(text string) []verifiers.Claim {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	var claims []verifiers.Claim
	seen := make(map[string]bool)

	add := func(t verifiers.ClaimType, quote, target string) {
		key := string(t) + "|" + target + "|" + quote
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
		})
	}

	for _, m := range reTestPass.FindAllStringIndex(text, -1) {
		add(verifiers.ClaimTestPass, text[m[0]:m[1]], "")
	}
	for _, m := range reCommitted.FindAllStringIndex(text, -1) {
		add(verifiers.ClaimCommitted, text[m[0]:m[1]], "")
	}
	for _, m := range rePushed.FindAllStringIndex(text, -1) {
		add(verifiers.ClaimPushed, text[m[0]:m[1]], "")
	}
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
		add(verifiers.ClaimCommandRan, text[m[0]:m[1]], "")
	}
	return claims
}

// HasActionProse reports whether prose contains action-oriented claims.
func HasActionProse(claims []verifiers.Claim) bool {
	for _, c := range claims {
		if c.Source == "prose" && verifiers.IsActionClaim(c.Type) {
			return true
		}
	}
	return false
}
