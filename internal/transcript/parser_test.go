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

func TestParseLinesToolResults(t *testing.T) {
	path := filepath.Join("..", "..", "test", "fixtures", "sample_transcripts", "cursor_tool_results.jsonl")
	lines, _, err := transcript.ParseLines(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	var calls []transcript.ToolCall
	var results []transcript.ToolResult
	for _, l := range lines {
		calls = append(calls, l.ToolCalls...)
		results = append(results, l.ToolResults...)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 tool_result, got %d", len(results))
	}
	if results[0].ToolUseID != "toolu_shell_01" || !results[0].IsError {
		t.Fatalf("unexpected result: %+v", results[0])
	}
	attached := transcript.AttachToolResults(calls, results)
	if len(attached) != 1 || attached[0].Result == "" || !attached[0].IsError {
		t.Fatalf("expected attached shell result, got %+v", attached)
	}
}

func TestProjectCwdFromTranscriptPath(t *testing.T) {
	path := "/Users/alice/.cursor/projects/Users-alice-code-snitch/agent-transcripts/uuid/uuid.jsonl"
	cwd := transcript.ProjectCwdFromTranscriptPath(path)
	if cwd != "/Users/alice/code/snitch" {
		t.Fatalf("got %q", cwd)
	}
}

func TestCursorProjectDirFromTranscriptPath(t *testing.T) {
	path := "/Users/alice/.cursor/projects/Users-alice-code-snitch/agent-transcripts/uuid/uuid.jsonl"
	dir := transcript.CursorProjectDirFromTranscriptPath(path)
	if dir == "" || !filepath.IsAbs(dir) {
		t.Fatalf("got %q", dir)
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
