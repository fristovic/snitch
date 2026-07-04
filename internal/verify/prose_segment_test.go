package verify_test

import (
	"testing"

	"github.com/fristovic/snitch/internal/verify"
	"github.com/fristovic/snitch/internal/verify/verifiers"
)

func TestExtractProseClaimsSegment(t *testing.T) {
	text := "Done.\n\n### Summary\nI've committed the changes."
	claims := verify.ExtractProseClaims(text)
	var committed *verifiers.Claim
	for i := range claims {
		if claims[i].Type == verifiers.ClaimCommitted {
			committed = &claims[i]
			break
		}
	}
	if committed == nil {
		t.Fatal("expected committed claim")
	}
	if committed.Segment != "recap" {
		t.Fatalf("segment=%q", committed.Segment)
	}
	if committed.Confidence != 1 {
		t.Fatalf("confidence=%d", committed.Confidence)
	}

	execClaims := verify.ExtractProseClaims("I updated `main.go`.")
	if len(execClaims) == 0 {
		t.Fatal("expected execution claims")
	}
	if execClaims[0].Segment != "execution" {
		t.Fatalf("segment=%q", execClaims[0].Segment)
	}
}
