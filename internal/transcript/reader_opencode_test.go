package transcript

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestOnDeviceOpenCodeDB(t *testing.T) {
	dbPath := filepath.Join(os.Getenv("HOME"), ".local/share/opencode/opencode.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Skip("on-device opencode db not present")
	}
	db, err := openOpenCodeDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	cwds, err := loadOpenCodeSessions(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(cwds) == 0 {
		t.Fatal("expected at least one session")
	}
	for sid, cwd := range cwds {
		turns, _, err := rebuildOpenCodeTurns(db, sid, cwd, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(turns) == 0 {
			continue // session may have no completed turns
		}
		if turns[0].ProjectPath != cwd {
			t.Fatalf("project path=%q want %q", turns[0].ProjectPath, cwd)
		}
		return
	}
	t.Skip("no session with completed turns on device")
}

// newFakeOpenCodeDB builds a minimal opencode.db with the schema the reader
// queries (session, message, part).
func newFakeOpenCodeDB(t *testing.T) (string, *sql.DB) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "opencode.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	for _, stmt := range []string{
		`CREATE TABLE session (id TEXT PRIMARY KEY, directory TEXT)`,
		`CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT, time_created INTEGER, data TEXT)`,
		`CREATE TABLE part (id TEXT PRIMARY KEY, message_id TEXT, session_id TEXT, time_created INTEGER, data TEXT)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	return path, db
}

func insertMsg(t *testing.T, db *sql.DB, id, sid string, ts int64, data string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO message (id, session_id, time_created, data) VALUES (?,?,?,?)`, id, sid, ts, data); err != nil {
		t.Fatal(err)
	}
}

func insertPart(t *testing.T, db *sql.DB, id, msgID, sid string, ts int64, data string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO part (id, message_id, session_id, time_created, data) VALUES (?,?,?,?,?)`, id, msgID, sid, ts, data); err != nil {
		t.Fatal(err)
	}
}

// A turn in progress (no finish=="stop") must NOT be emitted, and must NOT
// advance the cursor — it is picked up once it completes.
func TestOpenCodeInProgressTurnNotEmittedThenPickedUp(t *testing.T) {
	path, db := newFakeOpenCodeDB(t)
	if _, err := db.Exec(`INSERT INTO session (id, directory) VALUES ('s1', '/proj')`); err != nil {
		t.Fatal(err)
	}
	insertMsg(t, db, "m1", "s1", 1000, `{"role":"user"}`)
	insertPart(t, db, "p1", "m1", "s1", 1001, `{"type":"text","text":"do the thing"}`)
	insertMsg(t, db, "m2", "s1", 2000, `{"role":"assistant","finish":"tool-calls"}`)
	insertPart(t, db, "p2", "m2", "s1", 2001, `{"type":"tool","tool":"bash","callID":"c1","state":{"status":"completed","input":{"command":"ls"},"output":"files","metadata":{"exit":0}}}`)

	r := NewOpenCodeReader(path, nil)
	// Cursor starts at 0 (no seed): everything is fair game once complete.
	turns, err := r.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 0 {
		t.Fatalf("in-progress turn emitted: %+v", turns)
	}

	// The turn completes with NO newer parts — only a final stop message.
	insertMsg(t, db, "m3", "s1", 3000, `{"role":"assistant","finish":"stop"}`)
	insertPart(t, db, "p3", "m3", "s1", 3001, `{"type":"text","text":"done"}`)

	turns, err = r.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 1 {
		t.Fatalf("completed turn missed: %+v", turns)
	}
	turn := turns[0]
	if turn.UserText != "do the thing" || turn.ProjectPath != "/proj" || turn.Harness != "opencode" {
		t.Fatalf("wrong turn: %+v", turn)
	}
	if len(turn.ToolCalls) != 1 || turn.ToolCalls[0].Name != ToolShell || turn.ToolCalls[0].Result != "files" {
		t.Fatalf("wrong tool calls: %+v", turn.ToolCalls)
	}
	if turn.RunID == "" {
		t.Fatal("expected uuid RunID")
	}

	// Third poll: nothing new — the emitted turn must not repeat.
	turns, err = r.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 0 {
		t.Fatalf("turn emitted twice: %+v", turns)
	}
}

// A stop message with MULTIPLE parts must produce exactly one turn with all
// parts included — the end marker fires after the message's last row, not on
// the first part (which would fragment the turn).
func TestOpenCodeMultiPartStopMessage(t *testing.T) {
	path, db := newFakeOpenCodeDB(t)
	if _, err := db.Exec(`INSERT INTO session (id, directory) VALUES ('s1', '/proj')`); err != nil {
		t.Fatal(err)
	}
	insertMsg(t, db, "m1", "s1", 1000, `{"role":"user"}`)
	insertPart(t, db, "p1", "m1", "s1", 1001, `{"type":"text","text":"do it"}`)
	insertMsg(t, db, "m2", "s1", 2000, `{"role":"assistant","finish":"stop"}`)
	insertPart(t, db, "p2", "m2", "s1", 2001, `{"type":"tool","tool":"bash","callID":"c1","state":{"status":"completed","input":{"command":"ls"},"output":"ok","metadata":{"exit":0}}}`)
	insertPart(t, db, "p3", "m2", "s1", 2002, `{"type":"text","text":"first half"}`)
	insertPart(t, db, "p4", "m2", "s1", 2003, `{"type":"text","text":"second half"}`)

	r := NewOpenCodeReader(path, nil)
	turns, err := r.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d: %+v", len(turns), turns)
	}
	turn := turns[0]
	if turn.AssistantText != "first half\nsecond half" {
		t.Fatalf("assistant text fragmented: %q", turn.AssistantText)
	}
	if len(turn.ToolCalls) != 1 || turn.ToolCalls[0].Name != ToolShell {
		t.Fatalf("tool call lost: %+v", turn.ToolCalls)
	}
	// Timestamps come from DB event times, not poll wall-clock.
	if turn.StartedAt.UnixMilli() != 1001 || turn.FinishedAt.UnixMilli() != 2003 {
		t.Fatalf("timestamps not from DB events: started=%d finished=%d",
			turn.StartedAt.UnixMilli(), turn.FinishedAt.UnixMilli())
	}
}

// Messages without part rows must still count toward the cursor (msg time is
// folded into the turn's max event time).
func TestOpenCodeTurnWithNoParts(t *testing.T) {
	path, db := newFakeOpenCodeDB(t)
	if _, err := db.Exec(`INSERT INTO session (id, directory) VALUES ('s1', '/proj')`); err != nil {
		t.Fatal(err)
	}
	insertMsg(t, db, "m1", "s1", 1000, `{"role":"user"}`)
	insertPart(t, db, "p1", "m1", "s1", 1001, `{"type":"text","text":"hello"}`)
	insertMsg(t, db, "m2", "s1", 2000, `{"role":"assistant","finish":"stop"}`)
	insertPart(t, db, "p2", "m2", "s1", 2001, `{"type":"text","text":"hi"}`)
	// Second turn: stop message has NO parts at all.
	insertMsg(t, db, "m3", "s1", 3000, `{"role":"user"}`)
	insertPart(t, db, "p3", "m3", "s1", 3001, `{"type":"text","text":"again"}`)
	insertMsg(t, db, "m4", "s1", 4000, `{"role":"assistant","finish":"stop"}`)

	r := NewOpenCodeReader(path, nil)
	turns, err := r.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 2 {
		t.Fatalf("expected 2 turns, got %d: %+v", len(turns), turns)
	}

	// Incremental: only turns newer than the cursor emit on the next poll.
	insertMsg(t, db, "m5", "s1", 5000, `{"role":"user"}`)
	insertPart(t, db, "p5", "m5", "s1", 5001, `{"type":"text","text":"third"}`)
	insertMsg(t, db, "m6", "s1", 6000, `{"role":"assistant","finish":"stop"}`)
	turns, err = r.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 1 || turns[0].UserText != "third" {
		t.Fatalf("incremental poll wrong: %+v", turns)
	}
}
