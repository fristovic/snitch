package transcript_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fristovic/snitch/internal/transcript"
)

func TestLoadSubagentToolCalls(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "sess", "parent.jsonl")
	subDir := filepath.Join(dir, "sess", "subagents")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	subContent := `{"role":"assistant","message":{"content":[{"type":"tool_use","name":"Shell","input":{"command":"go test ./..."}}]}}
{"type":"turn_ended","status":"success"}
`
	if err := os.WriteFile(filepath.Join(subDir, "run-tests.jsonl"), []byte(subContent), 0o644); err != nil {
		t.Fatal(err)
	}

	start := time.Now().Add(-time.Minute)
	end := time.Now().Add(time.Minute)
	calls, err := transcript.LoadSubagentToolCalls(parent, start, end)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) == 0 {
		t.Fatal("expected merged subagent tool calls")
	}
	if calls[0].Name != "Shell" {
		t.Fatalf("got %q", calls[0].Name)
	}
	if calls[0].ToolUseID == "" || calls[0].ToolUseID[:9] != "subagent:" {
		t.Fatalf("expected subagent prefix, got %q", calls[0].ToolUseID)
	}
}

func TestBuildFileManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "foo.go")
	if err := os.WriteFile(path, []byte("package foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	calls := []transcript.ToolCall{{Name: "Write", Target: "foo.go"}}
	m := transcript.BuildFileManifest(dir, calls)
	if m["foo.go"] == "" {
		t.Fatal("expected hash for foo.go")
	}
}
