package transcript_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fristovic/snitch/internal/transcript"
)

// On-device regression tests: skip when the developer machine does not have
// the referenced session artifacts (CI-safe).
func TestOnDeviceClaudeSession(t *testing.T) {
	path := firstClaudeSessionWithTurnEnd(t)
	lines, _, err := transcript.ParseLinesWith(transcript.ClaudeParser{}, path, 0)
	if err != nil {
		t.Fatal(err)
	}
	var turnEnds int
	for _, l := range lines {
		if l.TurnEnded {
			turnEnds++
		}
	}
	if turnEnds == 0 {
		t.Fatal("expected at least one end_turn from real claude session")
	}
	r := transcript.ClaudePathResolver{}
	if cwd := r.ProjectCwd(path); cwd == "" {
		t.Fatal("expected non-empty ProjectCwd from real claude session")
	}
}

func TestOnDeviceCursorTurnEnded(t *testing.T) {
	path := firstCursorSessionWithTurnEnd(t)
	lines, _, err := transcript.ParseLinesWith(transcript.CursorParser{}, path, 0)
	if err != nil {
		t.Fatal(err)
	}
	var turnEnds int
	for _, l := range lines {
		if l.TurnEnded {
			turnEnds++
		}
	}
	if turnEnds == 0 {
		t.Fatal("expected turn_ended from real cursor session")
	}
}

func TestCodexPayloadParsing(t *testing.T) {
	p := transcript.CodexParser{}
	payloadLine := `{"type":"response_item","payload":{"type":"function_call","name":"shell","call_id":"c1","arguments":{"command":"ls"}}}`
	pl, ok := p.ParseLine(payloadLine)
	if !ok || len(pl.ToolCalls) != 1 || pl.ToolCalls[0].Name != "Shell" {
		t.Fatalf("failed to parse codex payload line: ok=%v calls=%+v", ok, pl.ToolCalls)
	}
	metaLine := `{"type":"session_meta","payload":{"cwd":"/tmp/proj"}}`
	pl, ok = p.ParseLine(metaLine)
	if !ok || pl.Cwd != "/tmp/proj" {
		t.Fatalf("session_meta cwd: ok=%v cwd=%q", ok, pl.Cwd)
	}
}

func TestPiOfficialNestedFormat(t *testing.T) {
	path := filepath.Join("..", "..", "test", "fixtures", "sample_transcripts", "pi_session.jsonl")
	lines, _, err := transcript.ParseLinesWith(transcript.PiParser{}, path, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) < 4 {
		t.Fatalf("expected parsed lines from official pi shape, got %d", len(lines))
	}
	var shells int
	for _, l := range lines {
		for _, tc := range l.ToolCalls {
			if tc.Name == "Shell" {
				shells++
			}
		}
	}
	if shells < 2 {
		t.Fatalf("expected bash toolCall + bashExecution as Shell, got %d", shells)
	}
}

func firstClaudeSessionWithTurnEnd(t *testing.T) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(os.Getenv("HOME"), ".claude", "projects", "*", "*.jsonl"))
	if err != nil || len(matches) == 0 {
		t.Skip("no claude sessions on device")
	}
	p := transcript.ClaudeParser{}
	r := transcript.ClaudePathResolver{}
	for _, path := range matches {
		if r.ProjectCwd(path) == "" {
			continue
		}
		lines, _, err := transcript.ParseLinesWith(p, path, 0)
		if err != nil {
			continue
		}
		for _, l := range lines {
			if l.TurnEnded {
				return path
			}
		}
	}
	t.Skip("no claude session with end_turn found")
	return ""
}

func firstCursorSessionWithTurnEnd(t *testing.T) string {
	t.Helper()
	root := filepath.Join(os.Getenv("HOME"), ".cursor", "projects")
	matches, err := filepath.Glob(filepath.Join(root, "*", "agent-transcripts", "*", "*.jsonl"))
	if err != nil || len(matches) == 0 {
		t.Skip("no cursor sessions on device")
	}
	p := transcript.CursorParser{}
	for _, path := range matches {
		lines, _, err := transcript.ParseLinesWith(p, path, 0)
		if err != nil {
			continue
		}
		for _, l := range lines {
			if l.TurnEnded {
				return path
			}
		}
	}
	t.Skip("no cursor session with turn_ended found")
	return ""
}
