package verifiers_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/transcript"
	"github.com/fristovic/snitch/internal/verify/verifiers"
)

func TestShellVerifierSyntaxOnly(t *testing.T) {
	v := verifiers.NewShellVerifier(config.ShellVerifierConfig{})
	res, err := v.Verify(verifiers.Claim{
		Type: "Shell", Source: "tool",
		Target: "go test ./...",
		Input:  map[string]any{"command": "go test ./..."},
	}, verifiers.VerifyContext{Cwd: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Accurate {
		t.Fatalf("expected syntax-only pass: %+v", res)
	}
}

func TestSubagentVerifierMissingDir(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "session.jsonl")
	v := &verifiers.SubagentVerifier{}
	res, err := v.Verify(verifiers.Claim{Type: "Task", Source: "tool"}, verifiers.VerifyContext{
		TranscriptPath: parent,
		ToolCalls:      []transcript.ToolCall{{Name: "Task"}},
		ObservedAt:     time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Accurate {
		t.Fatal("expected missing subagents dir to fail")
	}
}

func TestSubagentVerifierNonEmpty(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "sess", "sess.jsonl")
	subDir := filepath.Join(dir, "sess", "subagents")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "sub.jsonl"), []byte("line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v := &verifiers.SubagentVerifier{}
	res, err := v.Verify(verifiers.Claim{Type: "Task", Source: "tool"}, verifiers.VerifyContext{
		TranscriptPath: parent,
		ToolCalls:      []transcript.ToolCall{{Name: "Task"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Accurate {
		t.Fatalf("expected pass: %+v", res)
	}
}
