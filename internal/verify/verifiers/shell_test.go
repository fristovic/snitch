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
		Type: verifiers.ClaimToolShell, Source: "tool",
		Target: "go test ./...",
		Input:  map[string]any{"command": "go test ./..."},
	}, verifiers.VerifyContext{Cwd: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Epistemic != verifiers.EpistemicMissing {
		t.Fatalf("expected syntax-only missing (re-run disabled): %+v", res)
	}
}

func TestShellVerifierGitDiffDoesNotRequireCommitEvidence(t *testing.T) {
	v := verifiers.NewShellVerifier(config.ShellVerifierConfig{})
	res, err := v.Verify(verifiers.Claim{
		Type: verifiers.ClaimToolShell, Source: "tool",
		Target: "git diff README.md | head -80",
		Input:  map[string]any{"command": "git diff README.md | head -80"},
	}, verifiers.VerifyContext{
		Cwd: "/tmp",
		ToolCalls: []transcript.ToolCall{{
			Name:   "Shell",
			Target: "git diff README.md | head -80",
			Result: "diff --git a/README.md b/README.md\n",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Epistemic != verifiers.EpistemicSupported {
		t.Fatalf("git diff should not require commit evidence: %+v", res)
	}
	if res.GroundTruth == "claimed commit but no commit evidence" {
		t.Fatal("git diff incorrectly routed through commit verification")
	}
}

func TestShellVerifierGitStatusPasses(t *testing.T) {
	v := verifiers.NewShellVerifier(config.ShellVerifierConfig{})
	res, err := v.Verify(verifiers.Claim{
		Type: verifiers.ClaimToolShell, Source: "tool",
		Target: "git status",
		Input:  map[string]any{"command": "git status"},
	}, verifiers.VerifyContext{
		Cwd: "/tmp",
		ToolCalls: []transcript.ToolCall{{
			Name:   "Shell",
			Target: "git status",
			Result: "On branch main\n",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Epistemic != verifiers.EpistemicSupported {
		t.Fatalf("git status should pass when shell succeeds: %+v", res)
	}
}

func TestShellVerifierGitCommitRequiresEvidence(t *testing.T) {
	v := verifiers.NewShellVerifier(config.ShellVerifierConfig{})
	res, err := v.Verify(verifiers.Claim{
		Type: verifiers.ClaimToolShell, Source: "tool",
		Target: "git commit -m fix",
		Input:  map[string]any{"command": "git commit -m fix"},
	}, verifiers.VerifyContext{Cwd: "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Epistemic != verifiers.EpistemicMissing {
		t.Fatal("git commit without evidence should be missing")
	}
	if res.GroundTruth != "claimed commit but no commit evidence" {
		t.Fatalf("unexpected ground truth: %q", res.GroundTruth)
	}
}

func TestShellVerifierGitLogWithPushSubstringPasses(t *testing.T) {
	v := verifiers.NewShellVerifier(config.ShellVerifierConfig{})
	res, err := v.Verify(verifiers.Claim{
		Type: verifiers.ClaimToolShell, Source: "tool",
		Target: `git log --grep=push`,
		Input:  map[string]any{"command": `git log --grep=push`},
	}, verifiers.VerifyContext{
		Cwd: "/tmp",
		ToolCalls: []transcript.ToolCall{{
			Name:   "Shell",
			Target: `git log --grep=push`,
			Result: "commit abc\n",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Epistemic != verifiers.EpistemicSupported {
		t.Fatalf("git log should not be treated as git push: %+v", res)
	}
}

func TestSubagentVerifierMissingDir(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "session.jsonl")
	v := &verifiers.SubagentVerifier{}
	res, err := v.Verify(verifiers.Claim{Type: verifiers.ClaimToolTask, Source: "tool"}, verifiers.VerifyContext{
		TranscriptPath: parent,
		ToolCalls:      []transcript.ToolCall{{Name: "Task"}},
		ObservedAt:     time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Epistemic != verifiers.EpistemicMissing {
		t.Fatal("expected missing subagents dir")
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
	res, err := v.Verify(verifiers.Claim{Type: verifiers.ClaimToolTask, Source: "tool"}, verifiers.VerifyContext{
		TranscriptPath: parent,
		ToolCalls:      []transcript.ToolCall{{Name: "Task"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Epistemic != verifiers.EpistemicSupported {
		t.Fatalf("expected pass: %+v", res)
	}
}
