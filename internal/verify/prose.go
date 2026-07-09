package verify

import (
	"regexp"
	"strings"

	"github.com/fristovic/snitch/internal/verify/verifiers"
)

const filePathPattern = "[`" + `'"']?([\w./-]+(?:\.\w+)?)[` + `'"']?`

// claimPattern is one entry in the claim pattern registry. Patterns must be
// HIGH PRECISION: a false positive destroys user trust faster than a false
// negative. Every pattern MUST ship with example sentences that match and
// negative sentences that must not produce a claim — TestClaimPatternRegistry
// fails CI otherwise, so an untested pattern cannot be merged.
//
// See docs/extending-patterns.md for the contributor guide and review policy.
type claimPattern struct {
	// Type is the claim type this pattern extracts.
	Type verifiers.ClaimType
	// Regex matches the claim. Submatch TargetIdx (if > 0) captures the
	// claim target (e.g. a file path).
	Regex *regexp.Regexp
	// TargetIdx is the submatch index of the target, or 0 for no target.
	TargetIdx int
	// Examples are sentences that MUST produce this claim (min 3).
	Examples []string
	// Negatives are sentences that MUST NOT produce this claim — hypotheticals,
	// questions, instructions, reported speech (min 2).
	Negatives []string
}

// claimPatterns is the single registry of prose claim patterns. To add a
// pattern, append an entry here with Examples and Negatives; the meta-test
// enforces both. Suppression heuristics (figurative speech, historical
// references) live in prose_suppress.go keyed by claim type.
var claimPatterns = []claimPattern{
	{
		Type:  verifiers.ClaimTestPass,
		Regex: regexp.MustCompile(`(?i)\b(all tests pass(?:ed)?|tests pass(?:ed)?|tests are passing|test suite pass(?:ed)?|all tests green|verified (?:with|using) tests)\b`),
		Examples: []string{
			"All tests pass after the refactor.",
			"The tests are passing now.",
			"I ran the suite and verified with tests.",
		},
		Negatives: []string{
			"Do the tests pass on your machine?",
			"Once you fix the import, the tests passed in CI last week should pass again.",
		},
	},
	{
		Type:  verifiers.ClaimCommitted,
		Regex: regexp.MustCompile(`(?i)\b(i(?:'ve| have)? committed|committed (?:the |these )?changes|commit(?:ted)? (?:to|the) git)\b`),
		Examples: []string{
			"I've committed the fix.",
			"Committed the changes with a descriptive message.",
			"I committed to git and everything is saved.",
		},
		Negatives: []string{
			"You committed these changes yesterday, right?",
			"The team is committed to quality.",
		},
	},
	{
		Type:  verifiers.ClaimPushed,
		Regex: regexp.MustCompile(`(?i)\b(i(?:'ve| have)? pushed|pushed to (?:remote|origin|github))\b`),
		Examples: []string{
			"I've pushed the branch.",
			"Pushed to origin, CI should kick off.",
			"I pushed the fix just now.",
		},
		Negatives: []string{
			"This pushed for a cleaner design.",
			"The deadline pushed forward by a week.",
		},
	},
	{
		Type:      verifiers.ClaimFileCreated,
		Regex:     regexp.MustCompile("(?i)\\b(created|added|wrote) (?:the )?(?:file )?`([^`]+)`"),
		TargetIdx: 2,
		Examples: []string{
			"Created `internal/auth/session.go` with the session store.",
			"I added `config.yaml` to the repo root.",
			"Wrote `main_test.go` covering the parser.",
		},
		Negatives: []string{
			"Should I create `session.go` next?",
			"You could add `config.yaml` if you want overrides.",
		},
	},
	{
		Type:      verifiers.ClaimFileCreated,
		Regex:     regexp.MustCompile(`(?i)\b(created|added|wrote) (?:the )?(?:file )?(` + filePathPattern + `)`),
		TargetIdx: 3,
		Examples: []string{
			"Created internal/auth/session.go for the session store.",
			"I added scripts/build.sh to automate the release.",
			"Wrote parser_test.go with three cases.",
		},
		Negatives: []string{
			"We should have created docs/plan.md before starting.",
			"Would you like me to add helper.go?",
		},
	},
	{
		Type:      verifiers.ClaimFileModified,
		Regex:     regexp.MustCompile("(?i)\\b(updated|modified|edited|changed) (?:the )?(?:file )?`([^`]+)`"),
		TargetIdx: 2,
		Examples: []string{
			"Updated `watcher.go` to drain buffers on shutdown.",
			"I modified `config.go` to add the new key.",
			"Edited `README.md` with the install steps.",
		},
		Negatives: []string{
			"Have you updated `watcher.go` yet?",
			"If we modified `config.go`, the tests would break.",
		},
	},
	{
		Type:      verifiers.ClaimFileModified,
		Regex:     regexp.MustCompile(`(?i)\b(updated|modified|edited|changed) (?:the )?(?:file )?(` + filePathPattern + `)`),
		TargetIdx: 3,
		Examples: []string{
			"Updated watcher.go to fix the leak.",
			"Modified defaults.go with the new endpoint.",
			"I changed main.go to apply the log level.",
		},
		Negatives: []string{
			"Can you update main.go on your side?",
			"The docs say to edit config.yaml manually.",
		},
	},
	{
		Type:      verifiers.ClaimFileDeleted,
		Regex:     regexp.MustCompile("(?i)\\b(deleted|removed) (?:the )?(?:file )?`([^`]+)`"),
		TargetIdx: 2,
		Examples: []string{
			"Deleted `legacy_parser.go` since nothing calls it.",
			"Removed `old_config.yaml` from the repo.",
			"I deleted `tmp.txt` after the migration.",
		},
		Negatives: []string{
			"Should we delete `legacy_parser.go`?",
			"Careful — removing `config.yaml` would break the daemon.",
		},
	},
	{
		Type:      verifiers.ClaimFileDeleted,
		Regex:     regexp.MustCompile(`(?i)\b(deleted|removed) (?:the )?(?:file )?(` + filePathPattern + `)`),
		TargetIdx: 3,
		Examples: []string{
			"Deleted legacy_parser.go as part of the cleanup.",
			"Removed dead_code.go entirely.",
			"I deleted stale_test.go since the fixture moved.",
		},
		Negatives: []string{
			"Do you want me to delete helpers.go?",
			"Never remove migrations.sql without a backup.",
		},
	},
	{
		Type:  verifiers.ClaimCommandRan,
		Regex: regexp.MustCompile(`(?i)\b(?:i\s+)?(?:ran|executed)\s+(?:the\s+)?(?:\w+\s+)*command\b`),
		Examples: []string{
			"I ran the build command and it finished cleanly.",
			"Executed the migration command against the dev database.",
			"Ran the test command twice to be sure.",
		},
		Negatives: []string{
			"I ran the command in my head and it should work.",
			"You should have run the lint command first.",
		},
	},
	{
		Type:  verifiers.ClaimCommandSucceeded,
		Regex: regexp.MustCompile(`(?i)\b(?:(?:command|build)\s+(?:ran|run|completed|succeeded|successful(?:ly)?)|(?:successfully|cleanly)\s+ran)\b`),
		Examples: []string{
			"The build completed without warnings.",
			"The command succeeded on the first try.",
			"Everything successfully ran end to end.",
		},
		Negatives: []string{
			"If the build succeeded, we can tag the release.",
			"Tell me if the command ran on your machine.",
		},
	},
	{
		Type:  verifiers.ClaimStub,
		Regex: regexp.MustCompile(`(?i)\b(fully implemented|implementation (?:is )?complete|(?:all )?tasks? (?:are )?done|all done|nothing left to do|ready to ship)\b`),
		Examples: []string{
			"The feature is fully implemented and tested.",
			"Implementation is complete, ready for review.",
			"All tasks are done — ready to ship.",
		},
		Negatives: []string{
			"We're done with the planning phase, implementation comes next.",
			"Once the tests pass we're done building this feature.",
		},
	},
}

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

// extractProseFromSegment runs every registry pattern against one prose
// segment, applying per-type suppression heuristics and target validation.
func extractProseFromSegment(text, segment string) []verifiers.Claim {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	var claims []verifiers.Claim
	seen := make(map[string]bool)

	addWithContext := func(t verifiers.ClaimType, quote, target, sentence, context string) {
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
		if sentence == "" {
			sentence = quote
		}
		claims = append(claims, verifiers.Claim{
			Type:        t,
			Source:      "prose",
			Target:      target,
			Quote:       quote,
			Description: desc,
			Segment:     segment,
			Confidence:  scoreConfidence(t, target, quote, segment),
			Sentence:    sentence,
			Context:     context,
		})
	}

	for _, p := range claimPatterns {
		for _, m := range p.Regex.FindAllStringSubmatchIndex(text, -1) {
			start, end := m[0], m[1]
			if shouldSuppressClaim(p.Type, text, start, end) {
				continue
			}
			target := ""
			if p.TargetIdx > 0 && 2*p.TargetIdx+1 < len(m) && m[2*p.TargetIdx] >= 0 {
				target = strings.Trim(text[m[2*p.TargetIdx]:m[2*p.TargetIdx+1]], `"'`+"`")
			}
			quote := text[start:end]
			sentence, context := expandClaimWindow(text, start, end)
			addWithContext(p.Type, quote, target, sentence, context)
		}
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
