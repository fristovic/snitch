package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseTerminalFileCursorFormat(t *testing.T) {
	// Matches on-device Cursor terminal dumps: leading/trailing --- delimiters.
	content := `---
pid: 1
cwd: "/tmp/proj"
command: "echo hello"
started_at: 2026-06-18T11:47:43.808Z
---
hello world
---
exit_code: 0
ended_at: 2026-06-18T11:47:44.000Z
---
`
	dir := t.TempDir()
	path := filepath.Join(dir, "1.txt")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	rec, ok := parseTerminalFile(path)
	if !ok {
		t.Fatal("expected terminal file to parse")
	}
	if rec.Command != "echo hello" {
		t.Fatalf("command=%q", rec.Command)
	}
	if rec.Output != "hello world" {
		t.Fatalf("output=%q", rec.Output)
	}
	if rec.ExitCode != 0 {
		t.Fatalf("exit=%d", rec.ExitCode)
	}
	if rec.EndedAt.IsZero() {
		t.Fatal("expected ended_at")
	}
}

func TestCursorTerminalResolverOnDevice(t *testing.T) {
	termPath, transcriptPath, rec := firstParseableCursorTerminal(t)
	if !commandsMatch(rec.Command, rec.Command) {
		t.Fatal("terminal command empty")
	}
	start := rec.EndedAt.Add(-5 * time.Minute)
	end := rec.EndedAt.Add(5 * time.Minute)
	out, code, ok := CursorTerminalResolver{}.Resolve(
		ToolCall{Name: "Shell", Target: rec.Command},
		ShellOutputRequest{
			TranscriptPath: transcriptPath,
			Command:        rec.Command,
			StartedAt:      start,
			FinishedAt:     end,
		},
	)
	if !ok {
		t.Fatalf("expected terminal match on device (term=%s)", termPath)
	}
	if code != rec.ExitCode {
		t.Fatalf("exit=%d want %d", code, rec.ExitCode)
	}
	if len(out) == 0 {
		t.Fatal("expected non-empty output")
	}
}

func firstParseableCursorTerminal(t *testing.T) (termPath, transcriptPath string, rec terminalRecord) {
	t.Helper()
	root := filepath.Join(os.Getenv("HOME"), ".cursor", "projects")
	termMatches, err := filepath.Glob(filepath.Join(root, "*", "terminals", "*.txt"))
	if err != nil || len(termMatches) == 0 {
		t.Skip("no cursor terminal files on device")
	}
	transcriptMatches, _ := filepath.Glob(filepath.Join(root, "*", "agent-transcripts", "*", "*.jsonl"))

	for _, term := range termMatches {
		parsed, ok := parseTerminalFile(term)
		if !ok || strings.TrimSpace(parsed.Command) == "" || parsed.EndedAt.IsZero() {
			continue
		}
		projectDir := filepath.Dir(filepath.Dir(term))
		for _, tp := range transcriptMatches {
			if strings.HasPrefix(tp, projectDir) {
				return term, tp, parsed
			}
		}
	}
	t.Skip("no parseable cursor terminal with matching transcript")
	return "", "", terminalRecord{}
}
