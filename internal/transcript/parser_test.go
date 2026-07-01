package transcript_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fristovic/snitch/internal/transcript"
)

func TestParseLinesToolUse(t *testing.T) {
	path := filepath.Join("..", "..", "test", "fixtures", "sample_transcripts", "cursor_tools.jsonl")
	lines, off, err := transcript.ParseLines(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	if off == 0 {
		t.Fatal("expected bytes read")
	}
	var tools int
	var ended bool
	for _, l := range lines {
		if l.TurnEnded {
			ended = true
		}
		tools += len(l.ToolCalls)
	}
	if !ended {
		t.Fatal("expected turn_ended")
	}
	if tools != 2 {
		t.Fatalf("expected 2 tool calls, got %d", tools)
	}
}

func TestProjectCwdFromTranscriptPath(t *testing.T) {
	path := "/Users/alice/.cursor/projects/Users-alice-code-snitch/agent-transcripts/uuid/uuid.jsonl"
	cwd := transcript.ProjectCwdFromTranscriptPath(path)
	if cwd != "/Users/alice/code/snitch" {
		t.Fatalf("got %q", cwd)
	}
}

func TestSessionIDFromTranscriptPath(t *testing.T) {
	path := "/Users/alice/.cursor/projects/Users-alice/agent-transcripts/abc-123/abc-123.jsonl"
	if got := transcript.SessionIDFromTranscriptPath(path); got != "abc-123" {
		t.Fatalf("got %q", got)
	}
}

func TestParseMalformedLineSkipped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.jsonl")
	content := "{not json}\n" +
		`{"role":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"path":"x.go"}}]}}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, _, err := transcript.ParseLines(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || len(lines[0].ToolCalls) != 1 {
		t.Fatalf("unexpected lines: %+v", lines)
	}
}
