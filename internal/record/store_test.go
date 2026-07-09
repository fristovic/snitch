package record

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
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

func TestStoreMigratesLegacyRunsSchema(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "snitch.db")
	legacy, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	// Minimal v0.2-style runs table without session_id / project_path.
	_, err = legacy.Exec(`
		CREATE TABLE runs (
			id TEXT PRIMARY KEY,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			model TEXT,
			harness TEXT NOT NULL DEFAULT 'cursor',
			coverage TEXT,
			command TEXT,
			duration_ms INTEGER,
			output_hash TEXT,
			verdict TEXT NOT NULL DEFAULT 'unverified',
			max_severity INTEGER,
			claim_count INTEGER DEFAULT 0,
			verified_claims INTEGER DEFAULT 0,
			false_claims INTEGER DEFAULT 0,
			device_id TEXT NOT NULL
		);
		CREATE TABLE claims (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id TEXT NOT NULL,
			claim_type TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT 'prose',
			target TEXT NOT NULL DEFAULT '',
			claimed TEXT NOT NULL,
			actual TEXT,
			verified INTEGER DEFAULT 0,
			severity INTEGER DEFAULT 0,
			verifier TEXT,
			evidence TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE TABLE config (key TEXT PRIMARY KEY, value TEXT NOT NULL);
	`)
	if err != nil {
		t.Fatal(err)
	}
	_ = legacy.Close()

	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	err = store.InsertRun(Run{
		ID: "r-legacy", SessionID: "sess-1", ProjectPath: "/tmp/proj",
		TranscriptPath: "/tmp/t.jsonl", ToolCallCount: 2, DeviceID: "dev",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStoreMigratesLegacyClaimsSchema(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "snitch.db")
	legacy, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = legacy.Exec(`
		CREATE TABLE runs (
			id TEXT PRIMARY KEY,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			session_id TEXT,
			transcript_path TEXT,
			project_path TEXT,
			harness TEXT NOT NULL DEFAULT 'cursor',
			command TEXT,
			duration_ms INTEGER,
			output_hash TEXT,
			tool_call_count INTEGER DEFAULT 0,
			verdict TEXT NOT NULL DEFAULT 'unverified',
			max_severity INTEGER,
			claim_count INTEGER DEFAULT 0,
			verified_claims INTEGER DEFAULT 0,
			false_claims INTEGER DEFAULT 0,
			device_id TEXT NOT NULL,
			trace TEXT
		);
		CREATE TABLE claims (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			run_id TEXT NOT NULL,
			tool_call TEXT NOT NULL,
			target TEXT NOT NULL DEFAULT '',
			claimed TEXT NOT NULL,
			actual TEXT,
			verified INTEGER DEFAULT 0,
			severity INTEGER DEFAULT 0,
			verifier TEXT,
			evidence TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE TABLE config (key TEXT PRIMARY KEY, value TEXT NOT NULL);
	`)
	if err != nil {
		t.Fatal(err)
	}
	_ = legacy.Close()

	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, err = store.db.Exec(`INSERT INTO runs (id, device_id) VALUES ('r1', 'd')`)
	if err != nil {
		t.Fatal(err)
	}
	err = store.InsertClaims([]Claim{{
		RunID: "r1", ClaimType: "test_pass", Source: "prose", Claimed: "tests pass",
		Verified: -1, Severity: 3,
	}})
	if err != nil {
		t.Fatal(err)
	}
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

func TestInsertClaimsScrubsTrainingText(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	_ = store.InsertRun(Run{ID: "r1", Verdict: VerdictFail, DeviceID: "d"})
	secret := "sk-abcdefghijklmnopqrstuvwxyz012345"
	leak := "OPENAI_API_KEY=" + secret
	err = store.InsertClaims([]Claim{{
		RunID: "r1", ClaimType: "test_pass", Source: "prose",
		Claimed: "all tests pass " + leak,
		Actual:  "failed " + leak,
		ClaimSentence: "Sentence with " + leak + " inside.",
		ClaimContext:  "Context " + leak + " around.",
		Verified: -1, Severity: 3,
	}})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := store.GetClaimsByRun("r1")
	if err != nil || len(claims) != 1 {
		t.Fatalf("claims: %+v err=%v", claims, err)
	}
	c := claims[0]
	for _, field := range []string{c.Claimed, c.Actual, c.ClaimSentence, c.ClaimContext} {
		if strings.Contains(field, secret) {
			t.Fatalf("secret leaked in %q", field)
		}
		if !strings.Contains(field, "[REDACTED]") {
			t.Fatalf("expected scrub in %q", field)
		}
	}
}

func TestInsertClaims(t *testing.T) {
	dir := t.TempDir()
	store, _ := Open(dir)
	defer store.Close()
	_ = store.InsertRun(Run{ID: "r1", Verdict: VerdictPass, DeviceID: "d"})
	err := store.InsertClaims([]Claim{{
		RunID: "r1", ClaimType: "tool_write", Source: "tool", Target: "a.go", Claimed: "Write a.go",
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

func TestClaimStats(t *testing.T) {
	dir := t.TempDir()
	store, _ := Open(dir)
	defer store.Close()
	_ = store.InsertRun(Run{ID: "r1", Verdict: VerdictFail, DeviceID: "d"})
	_ = store.InsertRun(Run{ID: "r2", Verdict: VerdictPass, DeviceID: "d"})
	_ = store.InsertClaims([]Claim{{
		RunID: "r1", ClaimType: "test_pass", Source: "prose", Claimed: "tests pass",
		Verified: -1, Severity: 3,
	}})
	stats, err := store.ClaimStats()
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

func TestGetLatestTopFalseClaim(t *testing.T) {
	dir := t.TempDir()
	store, _ := Open(dir)
	defer store.Close()
	_ = store.InsertRun(Run{ID: "r1", ProjectPath: "/proj", SessionID: "s1", Verdict: VerdictFail, DeviceID: "d"})
	_ = store.InsertClaims([]Claim{
		{RunID: "r1", ClaimType: "file_modified", Source: "prose", Claimed: "updated", Verified: -1, Severity: 2},
		{RunID: "r1", ClaimType: "no_action", Source: "prose", Claimed: "updated", ClaimSentence: "I updated README.", Verified: -1, Severity: 3},
	})
	got, err := store.GetLatestTopFalseClaim()
	if err != nil || got == nil {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if got.ClaimType != "no_action" || got.Severity != 3 {
		t.Fatalf("expected top severity no_action, got %+v", got)
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
	claims, err := store.GetClaims(ClaimFilter{FalseClaimsOnly: true, ClaimType: "committed"})
	if err != nil || len(claims) != 1 {
		t.Fatalf("claims=%+v err=%v", claims, err)
	}
}

func TestGetRunsByProjectFilter(t *testing.T) {
	dir := t.TempDir()
	store, _ := Open(dir)
	defer store.Close()
	_ = store.InsertRun(Run{ID: "r1", ProjectPath: "/a", Verdict: VerdictPass, DeviceID: "d"})
	_ = store.InsertRun(Run{ID: "r2", ProjectPath: "/b", Verdict: VerdictPass, DeviceID: "d"})
	runs, err := store.GetRuns(RunFilter{ProjectPath: "/a", Limit: 10})
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
