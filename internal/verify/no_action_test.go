package verify

import (
	"strings"
	"testing"

	"github.com/fristovic/snitch/internal/verify/verifiers"
)

func TestSynthesizeNoActionFromActionProse(t *testing.T) {
	prose := []verifiers.Claim{{
		Type:     verifiers.ClaimCommitted,
		Source:   "prose",
		Quote:    "committed",
		Sentence: "I updated README and committed.",
		Context:  "Done. I updated README and committed. Thanks.",
		Segment:  "execution",
		Confidence: 3,
	}}
	c := synthesizeNoActionClaim(prose, "I updated README and committed.", reGenericActionProse)
	if c.Type != verifiers.ClaimNoAction {
		t.Fatalf("type: %s", c.Type)
	}
	if c.Sentence != "I updated README and committed." {
		t.Fatalf("sentence: %q", c.Sentence)
	}
	if c.Quote != "committed" {
		t.Fatalf("quote: %q", c.Quote)
	}
}

func TestSynthesizeNoActionFromGenericRegex(t *testing.T) {
	text := "Great news. I updated the docs today."
	c := synthesizeNoActionClaim(nil, text, reGenericActionProse)
	if c.Quote == "" {
		t.Fatal("expected quote from generic match")
	}
	if c.Sentence == "" {
		t.Fatal("expected sentence")
	}
	if !strings.Contains(strings.ToLower(c.Sentence), "updated") {
		t.Fatalf("sentence should include match: %q", c.Sentence)
	}
}

func TestEnrichClaimWindows(t *testing.T) {
	text := "I won't modify main.go. Then I edited it anyway."
	claims := []verifiers.Claim{{
		Type:   verifiers.ClaimSelfContradiction,
		Source: "consistency",
		Quote:  "won't modify main.go",
	}}
	enrichClaimWindows(text, claims)
	if claims[0].Sentence == "" {
		t.Fatal("expected sentence")
	}
	if !strings.Contains(claims[0].Sentence, "won't modify") {
		t.Fatalf("sentence: %q", claims[0].Sentence)
	}
}
