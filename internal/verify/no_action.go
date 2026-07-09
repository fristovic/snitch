package verify

import (
	"regexp"

	"github.com/fristovic/snitch/internal/verify/verifiers"
)

// synthesizeNoActionClaim builds a no_action claim with real flagged prose.
// Prefer copying Quote/Sentence/Context from the first matching action prose
// claim (deterministic); otherwise expand the first generic action-prose match.
func synthesizeNoActionClaim(proseClaims []verifiers.Claim, assistantText string, re *regexp.Regexp) verifiers.Claim {
	c := verifiers.Claim{
		Type:        verifiers.ClaimNoAction,
		Source:      "prose",
		Description: "action claimed in prose with zero tool calls",
	}
	if best, ok := firstActionProseClaim(proseClaims); ok {
		c.Quote = best.Quote
		c.Sentence = best.Sentence
		c.Context = best.Context
		c.Target = best.Target
		c.Segment = best.Segment
		c.Confidence = best.Confidence
		return c
	}
	if re == nil || assistantText == "" {
		return c
	}
	loc := re.FindStringIndex(assistantText)
	if loc == nil {
		return c
	}
	quote := assistantText[loc[0]:loc[1]]
	sentence, context := expandClaimWindow(assistantText, loc[0], loc[1])
	c.Quote = quote
	c.Sentence = sentence
	c.Context = context
	return c
}

func firstActionProseClaim(claims []verifiers.Claim) (verifiers.Claim, bool) {
	for _, c := range claims {
		if c.Source != "prose" {
			continue
		}
		if verifiers.IsLocalActionType(c.Type) {
			return c, true
		}
	}
	return verifiers.Claim{}, false
}
