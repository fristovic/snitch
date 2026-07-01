package cmd

import "testing"

func TestCycleVerdictFilter(t *testing.T) {
	f := filterState{}
	f = cycleVerdictFilter(f)
	if f.Verdict != "snitched" {
		t.Fatalf("expected snitched, got %s", f.Verdict)
	}
	f = cycleVerdictFilter(f)
	if f.Verdict != "all" || !f.ShowPasses {
		t.Fatalf("expected all, got %+v", f)
	}
}

func TestCycleClaimTypeFilter(t *testing.T) {
	f := filterState{}
	f = cycleClaimTypeFilter(f)
	if f.ClaimType != "test_pass" {
		t.Fatalf("expected test_pass, got %s", f.ClaimType)
	}
}
