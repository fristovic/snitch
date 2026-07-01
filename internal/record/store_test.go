package record

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreInsertAndQuery(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	run := Run{
		ID:          "run-1",
		SessionID:   "sess-1",
		ProjectPath: "/Users/test/proj",
		Verdict:     VerdictPass,
		DeviceID:    "dev-1",
	}
	if err := store.InsertRun(run); err != nil {
		t.Fatal(err)
	}

	runs, err := store.GetRuns(RunFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].ID != "run-1" {
		t.Fatalf("unexpected runs: %+v", runs)
	}
}

func TestStoreMigrationIdempotent(t *testing.T) {
	dir := t.TempDir()
	s1, _ := Open(dir)
	s1.Close()
	s2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
}

func TestEnsureDeviceID(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "snitch")
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	id1, err := store.EnsureDeviceID()
	if err != nil || id1 == "" {
		t.Fatal(err)
	}
	id2, _ := store.EnsureDeviceID()
	if id1 != id2 {
		t.Fatalf("device id changed: %s vs %s", id1, id2)
	}
	_ = os.RemoveAll(dir)
}

func TestInsertClaims(t *testing.T) {
	dir := t.TempDir()
	store, _ := Open(dir)
	defer store.Close()
	_ = store.InsertRun(Run{ID: "r1", Verdict: VerdictPass, DeviceID: "d"})
	err := store.InsertClaims([]Claim{{
		RunID: "r1", ClaimType: "Write", Source: "tool", Target: "a.go", Claimed: "Write a.go",
		Verified: 1, Severity: 0, CreatedAt: time.Now(),
	}})
	if err != nil {
		t.Fatal(err)
	}
	claims, _ := store.GetClaimsByRun("r1")
	if len(claims) != 1 {
		t.Fatalf("expected 1 claim, got %d", len(claims))
	}
}

func TestLieStats(t *testing.T) {
	dir := t.TempDir()
	store, _ := Open(dir)
	defer store.Close()
	_ = store.InsertRun(Run{ID: "r1", Verdict: VerdictFail, DeviceID: "d"})
	_ = store.InsertRun(Run{ID: "r2", Verdict: VerdictPass, DeviceID: "d"})
	_ = store.InsertClaims([]Claim{{
		RunID: "r1", ClaimType: "test_pass", Source: "prose", Claimed: "tests pass",
		Verified: -1, Severity: 3,
	}})
	stats, err := store.LieStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.SnitchedRuns != 1 {
		t.Fatalf("snitched=%d", stats.SnitchedRuns)
	}
	if stats.ByClaimType["test_pass"] != 1 {
		t.Fatalf("by type: %+v", stats.ByClaimType)
	}
}

func TestGetClaimsFilter(t *testing.T) {
	dir := t.TempDir()
	store, _ := Open(dir)
	defer store.Close()
	_ = store.InsertRun(Run{ID: "r1", ProjectPath: "/proj/a", SessionID: "s1", Verdict: VerdictFail, DeviceID: "d"})
	_ = store.InsertClaims([]Claim{{
		RunID: "r1", ClaimType: "committed", Source: "prose", Claimed: "committed",
		Verified: -1, Severity: 3,
	}})
	claims, err := store.GetClaims(ClaimFilter{LiesOnly: true, ClaimType: "committed"})
	if err != nil || len(claims) != 1 {
		t.Fatalf("claims=%+v err=%v", claims, err)
	}
}

func TestGetRunsByProject(t *testing.T) {
	dir := t.TempDir()
	store, _ := Open(dir)
	defer store.Close()
	_ = store.InsertRun(Run{ID: "r1", ProjectPath: "/a", Verdict: VerdictPass, DeviceID: "d"})
	_ = store.InsertRun(Run{ID: "r2", ProjectPath: "/b", Verdict: VerdictPass, DeviceID: "d"})
	runs, err := store.GetRunsByProject("/a", 10)
	if err != nil || len(runs) != 1 || runs[0].ID != "r1" {
		t.Fatalf("unexpected runs: %+v err %v", runs, err)
	}
}

func TestRunFilterSinceAndSearch(t *testing.T) {
	dir := t.TempDir()
	store, _ := Open(dir)
	defer store.Close()
	_ = store.InsertRun(Run{ID: "r1", Command: "fix the bug", Verdict: VerdictPass, DeviceID: "d"})
	runs, err := store.GetRuns(RunFilter{Search: "bug", Limit: 10})
	if err != nil || len(runs) != 1 {
		t.Fatalf("search: %+v err %v", runs, err)
	}
}
