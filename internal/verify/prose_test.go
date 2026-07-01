package verify_test

import (
	"strings"
	"testing"

	"github.com/fristovic/snitch/internal/verify"
	"github.com/fristovic/snitch/internal/verify/verifiers"
)

func TestExtractProseClaimsTestPass(t *testing.T) {
	claims := verify.ExtractProseClaims("I've verified with tests and all tests pass.")
	if len(claims) == 0 {
		t.Fatal("expected test_pass claim")
	}
	found := false
	for _, c := range claims {
		if c.Type == verifiers.ClaimTestPass {
			found = true
		}
	}
	if !found {
		t.Fatalf("claims: %+v", claims)
	}
}

func TestExtractProseClaimsNoFalsePositive(t *testing.T) {
	claims := verify.ExtractProseClaims("We should commit to this approach later.")
	for _, c := range claims {
		if c.Type == verifiers.ClaimCommitted {
			t.Fatalf("false positive commit claim: %+v", c)
		}
	}
}

func TestExtractProseClaimsFileCreated(t *testing.T) {
	claims := verify.ExtractProseClaims(`I created the file src/main.go for you.`)
	found := false
	for _, c := range claims {
		if c.Type == verifiers.ClaimFileCreated && strings.Contains(c.Target, "main.go") {
			found = true
		}
	}
	if !found {
		t.Fatalf("claims: %+v", claims)
	}
}
