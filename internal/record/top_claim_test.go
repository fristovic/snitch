package record

import "testing"

func TestSelectTopFalseClaim(t *testing.T) {
	claims := []Claim{
		{ClaimType: "stub", Verified: 1, Severity: 1},
		{ClaimType: "test_pass", Verified: -1, Severity: 3},
		{ClaimType: "committed", Verified: -1, Severity: 2},
	}
	top := SelectTopFalseClaim(claims)
	if top == nil || top.ClaimType != "test_pass" {
		t.Fatalf("got %+v", top)
	}
}
