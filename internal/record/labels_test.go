package record

import (
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// TestStoreLabels exercises the v1 data-flywheel label columns: setting a
// verdict, reading it back, querying unsynced shared labels, and marking synced.
func TestStoreLabels(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Seed two runs: one we'll label+share, one unlabeled.
	for _, id := range []string{"run-a", "run-b"} {
		if err := store.InsertRun(Run{
			ID: id, SessionID: "sess", Verdict: VerdictFail, DeviceID: "dev",
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.InsertClaims([]Claim{{
		RunID: "run-a", ClaimType: "test_pass", Source: "prose",
		Claimed: "all tests pass", Actual: "no test command ran",
		ClaimSentence: "Great news — all tests pass on main.",
		ClaimContext:  "Preface. Great news — all tests pass on main. Epilogue.",
		Verified: -1, Severity: 3,
	}}); err != nil {
		t.Fatal(err)
	}

	// Label run-a as correct and opted into sharing.
	if err := store.SetRunLabel("run-a", "correct", true, "fb-session-1"); err != nil {
		t.Fatal(err)
	}

	// Read it back.
	verdict, shared, ts, synced, err := store.GetRunLabel("run-a")
	if err != nil {
		t.Fatal(err)
	}
	if verdict != "correct" || !shared || synced {
		t.Fatalf("label readback wrong: verdict=%q shared=%v synced=%v", verdict, shared, synced)
	}
	if ts.IsZero() {
		t.Fatal("expected non-zero label timestamp")
	}

	// run-b should be unlabeled.
	v2, _, _, _, _ := store.GetRunLabel("run-b")
	if v2 != "" {
		t.Fatalf("expected run-b unlabeled, got %q", v2)
	}

	// Unsynced should include run-a (shared, not yet synced).
	unsynced, err := store.UnsyncedLabels(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(unsynced) != 1 || unsynced[0].RunID != "run-a" {
		t.Fatalf("expected run-a unsynced, got %+v", unsynced)
	}
	if unsynced[0].LabelVerdict != "correct" {
		t.Fatalf("expected label correct, got %q", unsynced[0].LabelVerdict)
	}
	if unsynced[0].ClaimSentence == "" || unsynced[0].Claimed == "" {
		t.Fatalf("expected training fields, got %+v", unsynced[0])
	}
	if unsynced[0].ClaimedTextHash == "" {
		t.Fatal("expected claimed_text_hash")
	}
	if unsynced[0].ClaimedTextHash != hashText(unsynced[0].ClaimSentence) {
		t.Fatalf("hash mismatch for sentence")
	}

	// Mark synced; next query should be empty.
	if err := store.MarkLabelsSynced([]string{"run-a"}); err != nil {
		t.Fatal(err)
	}
	again, _ := store.UnsyncedLabels(10)
	if len(again) != 0 {
		t.Fatalf("expected no unsynced after marking, got %+v", again)
	}

	// LabelTimestamp parse sanity: should be RFC3339-parsable.
	if _, err := time.Parse(time.RFC3339, ts.Format(time.RFC3339)); err != nil {
		t.Fatalf("timestamp not RFC3339: %v", err)
	}
}

func TestUnsyncedMissedClaimsTrainingFields(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if err := store.InsertRun(Run{
		ID: "run-m", SessionID: "sess", Verdict: VerdictPass, DeviceID: "dev",
		Harness: "cursor", Model: "gpt-test",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.AddMissedClaim("run-m", "all tests pass", "tests failed", true); err != nil {
		t.Fatal(err)
	}

	unsynced, err := store.UnsyncedMissedClaims(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(unsynced) != 1 {
		t.Fatalf("expected 1 missed claim, got %+v", unsynced)
	}
	l := unsynced[0]
	if l.LabelVerdict != "added" || l.ClaimType != "missed" {
		t.Fatalf("unexpected label meta: %+v", l)
	}
	if l.Claimed != "all tests pass" || l.Actual != "tests failed" {
		t.Fatalf("claimed/actual: %+v", l)
	}
	if l.ClaimSentence != l.Claimed {
		t.Fatalf("missed claim_sentence should fall back to claimed, got %q", l.ClaimSentence)
	}
	if l.ClaimedTextHash != hashText(l.ClaimSentence) {
		t.Fatal("hash mismatch")
	}
	if l.Harness != "cursor" || l.Model != "gpt-test" {
		t.Fatalf("harness/model: %+v", l)
	}

	if err := store.MarkMissedClaimsSynced([]int64{l.MissedID}); err != nil {
		t.Fatal(err)
	}
	again, _ := store.UnsyncedMissedClaims(10)
	if len(again) != 0 {
		t.Fatalf("expected empty after sync, got %+v", again)
	}
}

// TestStoreLabelMigrationIdempotent re-opens the DB to confirm the 005 ALTER
// statements are re-runnable without error (additive columns).
func TestStoreLabelMigrationIdempotent(t *testing.T) {
	dir := t.TempDir()
	s1, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s1.Close()
	s2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s2.Close()
}
