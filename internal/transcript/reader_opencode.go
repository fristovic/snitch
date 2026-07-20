package transcript

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/fristovic/snitch/internal/event"

	// modernc.org/sqlite is the pure-Go driver already used by internal/record.
	_ "modernc.org/sqlite"
)

// OpenCodeReader polls OpenCode's SQLite session database for completed turns.
//
// Unlike JSONL harnesses (which fsnotify watches), OpenCode stores sessions in
// a single SQLite DB. The reader opens the DB read-only and polls for new
// turns on an interval.
//
// Schema (confirmed from an actual opencode.db):
//
//	session(id, project_id, directory, title, ...)   // directory = project cwd
//	message(id, session_id, time_created INT ms, data TEXT)
//	  data = {"role":"user"|"assistant", "finish":"stop"|"tool-calls", "path":{"cwd":...}, ...}
//	part(id, message_id, session_id, time_created INT ms, data TEXT)
//	  data.type ∈ {text, tool, step-start, step-finish, reasoning, file}
//	  tool part: {type:"tool", tool:"bash", callID, state:{status, input, output, metadata:{exit}}}
//
// Turn boundary: an assistant message whose data.finish == "stop" completes a
// turn. (finish == "tool-calls" means tool calls follow.) A new user message
// also starts a new turn.
//
// Incremental cursor: lastEmitted tracks, per session, the newest event time
// (message OR part time_created) of turns that were actually emitted. It only
// advances past completed turns, so an in-progress turn is never skipped when
// it later completes, and a completed turn is never emitted twice.
type OpenCodeReader struct {
	dbPath string
	bus    *event.Bus
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc

	// lastEmitted is the per-session cursor: the max event time (ms) covered
	// by turns already emitted (or present before the reader started).
	lastEmitted map[string]int64
}

// NewOpenCodeReader creates a reader for the OpenCode DB at dbPath.
func NewOpenCodeReader(dbPath string, bus *event.Bus) *OpenCodeReader {
	return &OpenCodeReader{
		dbPath:      dbPath,
		bus:         bus,
		lastEmitted: make(map[string]int64),
	}
}

// Start seeds session cursors and starts the poll goroutine when a bus is set.
func (r *OpenCodeReader) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	r.ctx = ctx
	r.cancel = cancel
	if err := r.seed(); err != nil {
		return err
	}
	if r.bus != nil {
		go r.pollLoop()
	}
	return nil
}

// Stop shuts down the reader.
func (r *OpenCodeReader) Stop() error {
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}

func (r *OpenCodeReader) pollLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			turns, err := r.Poll()
			if err != nil {
				continue
			}
			for _, t := range turns {
				PublishTurnCompleted(r.bus, t)
			}
		}
	}
}

// seed records the newest event time (message or part) per session so Poll
// only emits turns that complete after the reader started. Sessions whose
// history ended before startup are skipped entirely.
func (r *OpenCodeReader) seed() error {
	db, err := openOpenCodeDB(r.dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	rows, err := db.Query(`
		SELECT session_id, MAX(t) FROM (
			SELECT session_id, time_created AS t FROM message
			UNION ALL
			SELECT session_id, time_created AS t FROM part
		) GROUP BY session_id`)
	if err != nil {
		return err // DB may not exist yet — benign on a fresh install
	}
	defer rows.Close()
	r.mu.Lock()
	for rows.Next() {
		var sid string
		var last sql.NullInt64
		if err := rows.Scan(&sid, &last); err != nil {
			continue
		}
		if last.Valid {
			r.lastEmitted[sid] = last.Int64
		}
	}
	r.mu.Unlock()
	return rows.Err()
}

// openOpenCodeDB opens the OpenCode DB read-only with WAL-friendly pragmas.
func openOpenCodeDB(dbPath string) (*sql.DB, error) {
	return sql.Open("sqlite", dbPath+"?mode=ro&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
}

// Poll returns turns completed since the previous poll. The per-session cursor
// advances only past turns actually emitted, so in-progress turns are picked
// up on a later poll once they complete.
func (r *OpenCodeReader) Poll() ([]TurnCompleted, error) {
	db, err := openOpenCodeDB(r.dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// session id → project cwd (from session.directory).
	cwds, err := loadOpenCodeSessions(db)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	snap := make(map[string]int64, len(r.lastEmitted))
	for k, v := range r.lastEmitted {
		snap[k] = v
	}
	r.mu.Unlock()

	var turns []TurnCompleted
	for sid, cwd := range cwds {
		since := snap[sid]
		rebuilt, emittedMax, err := rebuildOpenCodeTurns(db, sid, cwd, since)
		if err != nil {
			slog.Debug("opencode rebuild failed", "session", sid, "err", err)
			continue
		}
		turns = append(turns, rebuilt...)
		if emittedMax > since {
			r.mu.Lock()
			if emittedMax > r.lastEmitted[sid] {
				r.lastEmitted[sid] = emittedMax
			}
			r.mu.Unlock()
		}
	}
	return turns, nil
}

// loadOpenCodeSessions returns session_id → directory (project cwd).
func loadOpenCodeSessions(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query(`SELECT id, directory FROM session`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var id, dir string
		if err := rows.Scan(&id, &dir); err != nil {
			continue
		}
		out[id] = dir
	}
	return out, rows.Err()
}

// openCodeSessionLines translates a session's message+part rows into the
// normalized ParsedLine stream that the shared turnAssembler consumes:
//
//   - a user message row → user line with TurnEnded (starts the next turn,
//     Pi-style carry)
//   - assistant text/tool part rows → assistant lines
//   - a message with finish=="stop" → an explicit end-marker line after its
//     rows (Cursor-style marker)
//
// Every line carries the DB event time so turn timestamps and the poll
// cursor reflect reality rather than poll wall-clock time. When sinceMs > 0,
// only messages created after that timestamp are loaded.
func openCodeSessionLines(db *sql.DB, sessionID string, sinceMs int64) ([]ParsedLine, error) {
	// Join message → part, extract role/finish from message JSON data. Order by
	// message time then part time so parts group under their parent message.
	query := `
		SELECT m.id,
		       json_extract(m.data,'$.role'),
		       json_extract(m.data,'$.finish'),
		       m.time_created,
		       p.id, p.data, p.time_created
		FROM message m
		LEFT JOIN part p ON p.message_id = m.id
		WHERE m.session_id = ?`
	args := []any{sessionID}
	if sinceMs > 0 {
		query += ` AND m.time_created > ?`
		args = append(args, sinceMs)
	}
	query += ` ORDER BY m.time_created ASC, p.time_created ASC`
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		lines     []ParsedLine
		prevMsgID string
		prevStop  bool
		maxMs     int64 // running max event time (rows are ordered ascending)
	)
	// endMarker closes the previous message's turn when that message carried
	// finish=="stop". Emitted on message change and after the final row.
	endMarker := func() {
		if prevStop {
			lines = append(lines, ParsedLine{TurnEnded: true, Timestamp: msToTime(maxMs)})
			prevStop = false
		}
	}

	for rows.Next() {
		var msgID, role, finish sql.NullString
		var msgTimeMs sql.NullInt64
		var partID, partData sql.NullString
		var partTimeMs sql.NullInt64
		if err := rows.Scan(&msgID, &role, &finish, &msgTimeMs, &partID, &partData, &partTimeMs); err != nil {
			continue
		}
		if msgID.String != prevMsgID {
			endMarker()
			prevMsgID = msgID.String
			prevStop = finish.String == "stop"
		}
		eventMs := msgTimeMs.Int64
		if partTimeMs.Int64 > eventMs {
			eventMs = partTimeMs.Int64
		}
		if eventMs > maxMs {
			maxMs = eventMs
		}

		switch role.String {
		case "user":
			pl := ParsedLine{Role: "user", TurnEnded: true, Timestamp: msToTime(eventMs)}
			if partID.Valid && partData.String != "" {
				if pt, ptext, _ := decodeOpenCodePart(partData.String); pt == "text" {
					pl.Text = ptext
				}
			}
			lines = append(lines, pl)

		case "assistant":
			pl := ParsedLine{Role: "assistant", Timestamp: msToTime(eventMs)}
			if partID.Valid && partData.String != "" {
				pt, ptext, tc := decodeOpenCodePart(partData.String)
				switch pt {
				case "text":
					pl.Text = ptext
				case "tool":
					if tc.Name != "" {
						pl.ToolCalls = []ToolCall{tc}
					}
				}
			}
			if pl.Text != "" || len(pl.ToolCalls) > 0 {
				lines = append(lines, pl)
			}
		}
	}
	endMarker()
	return lines, rows.Err()
}

// rebuildOpenCodeTurns reconstructs completed turns from a session by feeding
// its rows through the same turnAssembler the JSONL watcher uses — OpenCode
// has no bespoke boundary logic. Only completed turns whose newest event time
// is after `since` (ms epoch) are emitted; the in-progress trailing buffer is
// discarded and re-assembled by a later poll once it completes. Returns the
// emitted turns and the max event time covered by them (0 when none).
func rebuildOpenCodeTurns(db *sql.DB, sessionID, projectCwd string, since int64) ([]TurnCompleted, int64, error) {
	lines, err := openCodeSessionLines(db, sessionID, since)
	if err != nil {
		return nil, 0, err
	}

	a := newTurnAssembler(OpenCodePathResolver{}, "(opencode)")
	var (
		turns      []TurnCompleted
		emittedMax int64
	)
	for _, line := range lines {
		buf := a.Feed(line)
		if buf == nil {
			continue
		}
		endMs := buf.lastWriteAt.UnixMilli()
		if endMs <= since {
			continue
		}
		toolCalls := AttachToolResults(buf.toolCalls, buf.toolResults)
		turns = append(turns, TurnCompleted{
			RunID:          uuid.NewString(),
			SessionID:      sessionID,
			TranscriptPath: "(opencode)",
			ProjectPath:    projectCwd,
			Harness:        "opencode",
			Model:          buf.model,
			UserText:       buf.userText,
			AssistantText:  buf.assistantText.String(),
			ToolCalls:      toolCalls,
			StartedAt:      buf.startedAt,
			FinishedAt:     buf.lastWriteAt,
			EndHEAD:        GitHEAD(projectCwd),
			FileManifest:   BuildFileManifest(projectCwd, toolCalls),
		})
		if endMs > emittedMax {
			emittedMax = endMs
		}
	}
	return turns, emittedMax, nil
}

// decodeOpenCodePart decodes a part row's data JSON into its text and/or
// ToolCall. Returns partType ("text"|"tool"|"reasoning"|""), the text (if any),
// and a ToolCall (for tool parts).
//
// Confirmed part.data shapes:
//
//	text:        {"type":"text","text":"..."}
//	tool:        {"type":"tool","tool":"bash","callID":"...",
//	              "state":{"status":"completed","input":{"command":"..."},
//	                       "output":"...","metadata":{"exit":0}}}
//	step-start:  {"type":"step-start"}
//	step-finish: {"type":"step-finish","reason":"...","tokens":{...}}
//	reasoning:   {"type":"reasoning","text":"..."}
func decodeOpenCodePart(dataJSON string) (ptype, text string, tc ToolCall) {
	if dataJSON == "" {
		return "", "", tc
	}
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(dataJSON), &head); err != nil {
		return "", "", tc
	}
	ptype = head.Type

	switch head.Type {
	case "text", "reasoning":
		var p struct {
			Text string `json:"text"`
		}
		_ = json.Unmarshal([]byte(dataJSON), &p)
		return ptype, p.Text, tc

	case "tool":
		var tool struct {
			Tool   string `json:"tool"`
			CallID string `json:"callID"`
			State  struct {
				Status   string          `json:"status"`
				Input    json.RawMessage `json:"input"`
				Output   string          `json:"output"`
				Metadata struct {
					Exit int `json:"exit"`
				} `json:"metadata"`
			} `json:"state"`
		}
		_ = json.Unmarshal([]byte(dataJSON), &tool)
		tc = NewToolCall(tool.Tool, nil)
		tc.ToolUseID = tool.CallID
		if len(tool.State.Input) > 0 {
			_ = json.Unmarshal(tool.State.Input, &tc.Input)
		}
		tc.Target = deriveTarget(tc)
		if tool.State.Output != "" {
			tc.Result = tool.State.Output
		}
		if tool.State.Metadata.Exit != 0 {
			tc.IsError = true
		}
		return ptype, "", tc
	}

	// step-start, step-finish, file, unknown: no extraction.
	return ptype, "", tc
}

// msToTime converts a Unix millisecond epoch (OpenCode's time_created) to time.Time.
func msToTime(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}
