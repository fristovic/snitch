package transcript_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fristovic/snitch/internal/transcript"
)

func TestParseLinesToolUse(t *testing.T) {
	path := filepath.Join("..", "..", "test", "fixtures", "sample_transcripts", "cursor_tools.jsonl")
	lines, off, err := transcript.ParseLinesWith(transcript.CursorParser{}, path, 0)
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
	lines, _, err := transcript.ParseLinesWith(transcript.CursorParser{}, path, 0)
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

func TestCursorPathResolver(t *testing.T) {
	r := transcript.CursorPathResolver{}
	path := "/Users/alice/.cursor/projects/Users-alice-code-snitch/agent-transcripts/abc-123/abc-123.jsonl"
	if cwd := r.ProjectCwd(path); cwd != "/Users/alice/code/snitch" {
		t.Fatalf("ProjectCwd got %q", cwd)
	}
	if dir := r.ProjectDir(path); dir == "" || !filepath.IsAbs(dir) {
		t.Fatalf("ProjectDir got %q", dir)
	}
	if got := r.SessionID(path); got != "abc-123" {
		t.Fatalf("SessionID got %q", got)
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
	lines, _, err := transcript.ParseLinesWith(transcript.CursorParser{}, path, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || len(lines[0].ToolCalls) != 1 {
		t.Fatalf("unexpected lines: %+v", lines)
	}
}
