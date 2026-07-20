package record

import "testing"

func TestSelectTopFalseClaim(t *testing.T) {
	claims := []Claim{
		{ClaimType: "stub", Epistemic: "supported", Severity: 1},
		{ClaimType: "test_pass", Epistemic: "contradicted", Severity: 3},
		{ClaimType: "committed", Epistemic: "contradicted", Severity: 2},
	}
	top := SelectTopFalseClaim(claims)
	if top == nil || top.ClaimType != "test_pass" {
		t.Fatalf("got %+v", top)
	}
}
