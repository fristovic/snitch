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
