package verifiers

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fristovic/snitch/internal/transcript"
)

// ShellOutputForCommand returns captured output for a shell tool call.
// Resolution order: tool_result on the call, then Cursor terminal files.
func ShellOutputForCommand(tc transcript.ToolCall, ctx VerifyContext) (output string, exitCode int, found bool) {
	if tc.Result != "" || tc.IsError {
		code := 0
		if tc.IsError {
			code = 1
		}
		return tc.Result, code, true
	}
	cmd := shellCommand(tc)
	if cmd == "" {
		return "", 0, false
	}
	return terminalOutputForCommand(ctx.TranscriptPath, cmd, ctx.StartedAt, ctx.FinishedAt)
}

func terminalOutputForCommand(transcriptPath, command string, start, end time.Time) (string, int, bool) {
	projectDir := transcript.CursorProjectDirFromTranscriptPath(transcriptPath)
	if projectDir == "" {
		return "", 0, false
	}
	termDir := filepath.Join(projectDir, "terminals")
	entries, err := os.ReadDir(termDir)
	if err != nil {
		return "", 0, false
	}
	command = strings.TrimSpace(command)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		rec, ok := parseTerminalFile(filepath.Join(termDir, e.Name()))
		if !ok {
			continue
		}
		if !commandsMatch(rec.Command, command) {
			continue
		}
		if !timeInWindow(rec.EndedAt, start, end) {
			continue
		}
		return rec.Output, rec.ExitCode, true
	}
	return "", 0, false
}

type terminalRecord struct {
	Command  string
	Output   string
	ExitCode int
	EndedAt  time.Time
}

func parseTerminalFile(path string) (terminalRecord, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return terminalRecord{}, false
	}
	text := string(data)
	parts := strings.Split(text, "\n---\n")
	if len(parts) < 3 {
		return terminalRecord{}, false
	}
	meta := parts[0]
	body := strings.TrimSpace(parts[1])
	footer := parts[len(parts)-1]

	var rec terminalRecord
	rec.Output = body
	for _, line := range strings.Split(meta, "\n") {
		if strings.HasPrefix(line, "command: ") {
			rec.Command = strings.Trim(strings.TrimPrefix(line, "command: "), `"`)
		}
		if strings.HasPrefix(line, "started_at: ") {
			if t, err := time.Parse(time.RFC3339Nano, strings.TrimPrefix(line, "started_at: ")); err == nil {
				rec.EndedAt = t
			}
		}
	}
	for _, line := range strings.Split(footer, "\n") {
		if strings.HasPrefix(line, "exit_code: ") {
			var code int
			if _, err := parseIntPrefix(line, "exit_code: ", &code); err == nil {
				rec.ExitCode = code
			}
		}
		if strings.HasPrefix(line, "ended_at: ") {
			if t, err := time.Parse(time.RFC3339Nano, strings.TrimPrefix(line, "ended_at: ")); err == nil {
				rec.EndedAt = t
			}
		}
	}
	if rec.Command == "" {
		return terminalRecord{}, false
	}
	return rec, true
}

func parseIntPrefix(line, prefix string, out *int) (bool, error) {
	val := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	n := 0
	for _, c := range val {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	*out = n
	return true, nil
}

func commandsMatch(a, b string) bool {
	a = normalizeCommand(a)
	b = normalizeCommand(b)
	if a == b {
		return true
	}
	return strings.Contains(a, b) || strings.Contains(b, a)
}

func normalizeCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	cmd = strings.Trim(cmd, `"'`)
	return strings.Join(strings.Fields(cmd), " ")
}

func timeInWindow(t, start, end time.Time) bool {
	if t.IsZero() || start.IsZero() || end.IsZero() {
		return true
	}
	// Allow terminal files that finished shortly after the turn ended.
	return !t.Before(start.Add(-2*time.Minute)) && !t.After(end.Add(5*time.Minute))
}

// ParseTestOutput inspects command output for pass/fail signals.
func ParseTestOutput(output string) (passed bool, found bool) {
	lower := strings.ToLower(output)
	if strings.Contains(lower, "build failed") ||
		strings.Contains(lower, "--- fail") ||
		strings.Contains(lower, "test failed") ||
		strings.Contains(lower, "failures") ||
		strings.Contains(lower, "\tfail") ||
		strings.HasPrefix(lower, "fail") ||
		(strings.Contains(lower, "failed") && !strings.Contains(lower, "0 failed")) {
		return false, true
	}
	if strings.Contains(lower, "ok\t") ||
		strings.Contains(lower, "passed") ||
		strings.Contains(lower, "all tests passed") ||
		(strings.Contains(lower, "exit_code: 0") && strings.Contains(lower, "test")) {
		return true, true
	}
	return false, false
}

// IsStubBody reports whether file contents look like an unimplemented placeholder.
func IsStubBody(body string) bool {
	trim := strings.TrimSpace(body)
	if trim == "" || trim == "pass" || trim == "..." {
		return true
	}
	lower := strings.ToLower(trim)
	stubs := []string{
		`panic("todo")`, `panic('todo')`, `panic("not implemented")`,
		"notimplementederror", `throw new error("not implemented")`,
		"// todo", "# todo", "todo:", "fixme", "unimplemented",
	}
	for _, s := range stubs {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}
