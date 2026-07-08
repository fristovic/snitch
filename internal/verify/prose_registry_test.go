package verify

import (
	"testing"

	"github.com/fristovic/snitch/internal/verify/verifiers"
)

// TestClaimPatternRegistry enforces the contribution contract for claim
// patterns: every pattern MUST carry at least 3 example sentences that
// produce its claim and at least 2 negative sentences that don't. A pattern
// without working examples/negatives cannot be merged — this test fails CI.
//
// See docs/extending-patterns.md.
func TestClaimPatternRegistry(t *testing.T) {
	if len(claimPatterns) == 0 {
		t.Fatal("claim pattern registry is empty")
	}
	for i, p := range claimPatterns {
		p := p
		name := string(p.Type)
		t.Run(name, func(t *testing.T) {
			if p.Regex == nil {
				t.Fatalf("pattern %d (%s): nil regex", i, name)
			}
			if len(p.Examples) < 3 {
				t.Fatalf("pattern %d (%s): needs at least 3 Examples, has %d — every pattern must prove it matches real claims", i, name, len(p.Examples))
			}
			if len(p.Negatives) < 2 {
				t.Fatalf("pattern %d (%s): needs at least 2 Negatives, has %d — every pattern must prove it rejects hypotheticals/questions", i, name, len(p.Negatives))
			}
			for _, ex := range p.Examples {
				if !producesClaim(ex, p.Type) {
					t.Errorf("pattern %d (%s): example did not produce a %s claim:\n  %q", i, name, name, ex)
				}
			}
			for _, neg := range p.Negatives {
				if patternProducesClaim(p, neg) {
					t.Errorf("pattern %d (%s): negative sentence produced a claim (false positive):\n  %q", i, name, neg)
				}
			}
		})
	}
}

// producesClaim runs the FULL extraction pipeline (all patterns + suppression
// + target validation) and reports whether a claim of type t comes out.
func producesClaim(text string, t verifiers.ClaimType) bool {
	for _, c := range ExtractProseClaims(text) {
		if c.Type == t {
			return true
		}
	}
	return false
}

// patternProducesClaim checks whether ONE pattern (with the shared
// suppression + target validation applied) fires on text. Negatives are
// checked per-pattern so a sentence rejected by pattern A can't be masked by
// pattern B matching something else.
func patternProducesClaim(p claimPattern, text string) bool {
	for _, m := range p.Regex.FindAllStringSubmatchIndex(text, -1) {
		start, end := m[0], m[1]
		if shouldSuppressClaim(p.Type, text, start, end) {
			continue
		}
		if p.TargetIdx > 0 && 2*p.TargetIdx+1 < len(m) && m[2*p.TargetIdx] >= 0 {
			target := verifiers.NormalizePathToken(text[m[2*p.TargetIdx]:m[2*p.TargetIdx+1]])
			if isFileClaim(p.Type) && !verifiers.LooksLikePath(target) {
				continue
			}
		}
		return true
	}
	return false
}
