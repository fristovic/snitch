package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CursorTerminalResolver resolves shell output from Cursor's per-project
// terminals/*.txt dump files. Cursor is the only harness with a terminal-file
// fallback; all other harnesses embed shell output inline in tool_result and
// use NoopShellOutputResolver.
type CursorTerminalResolver struct{}

// Resolve finds a Cursor terminal record matching the command within the turn
// time window. Returns found=false if no terminal file matches.
func (CursorTerminalResolver) Resolve(tc ToolCall, req ShellOutputRequest) (string, int, bool) {
	if req.TranscriptPath == "" || req.Command == "" {
		return "", 0, false
	}
	projectDir := CursorPathResolver{}.ProjectDir(req.TranscriptPath)
	if projectDir == "" {
		return "", 0, false
	}
	termDir := filepath.Join(projectDir, "terminals")
	entries, err := os.ReadDir(termDir)
	if err != nil {
		return "", 0, false
	}
	command := strings.TrimSpace(req.Command)
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
		if !timeInWindow(rec.EndedAt, req.StartedAt, req.FinishedAt) {
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

// parseTerminalFile decodes a Cursor terminal dump: a YAML-ish metadata header
// (pid/cwd/command/started_at), the command output body, and a footer with
// exit_code/ended_at, each delimited by --- lines.
func parseTerminalFile(path string) (terminalRecord, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return terminalRecord{}, false
	}
	sections := terminalSections(string(data))
	if len(sections) < 2 {
		return terminalRecord{}, false
	}

	metaIdx, footerIdx := -1, -1
	for i, sec := range sections {
		if strings.Contains(sec, "command:") && metaIdx < 0 {
			metaIdx = i
		}
		if strings.Contains(sec, "exit_code:") || strings.Contains(sec, "ended_at:") {
			footerIdx = i
		}
	}
	if metaIdx < 0 || footerIdx <= metaIdx {
		return terminalRecord{}, false
	}

	meta := sections[metaIdx]
	footer := sections[footerIdx]
	body := ""
	if footerIdx > metaIdx+1 {
		body = strings.TrimSpace(strings.Join(sections[metaIdx+1:footerIdx], "\n---\n"))
	}

	var rec terminalRecord
	rec.Output = body
	for _, line := range strings.Split(meta, "\n") {
		if strings.HasPrefix(line, "command: ") {
			rec.Command = unescapeTerminalCommand(strings.TrimPrefix(line, "command: "))
		}
	}
	for _, line := range strings.Split(footer, "\n") {
		if strings.HasPrefix(line, "exit_code: ") {
			rec.ExitCode = parseIntPrefix(line, "exit_code: ")
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

// terminalSections splits a Cursor terminal dump on --- delimiters, ignoring
// leading/trailing empty sections produced by files that start or end with ---.
func terminalSections(text string) []string {
	var out []string
	for _, p := range strings.Split(text, "\n---\n") {
		p = strings.TrimSpace(p)
		p = strings.TrimPrefix(p, "---")
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// unescapeTerminalCommand normalizes command strings from Cursor terminal
// metadata, which JSON/YAML-escape inner quotes (\"path\").
func unescapeTerminalCommand(raw string) string {
	cmd := strings.TrimSpace(raw)
	cmd = strings.Trim(cmd, `"'`)
	cmd = strings.ReplaceAll(cmd, `\"`, `"`)
	return cmd
}

func parseIntPrefix(line, prefix string) int {
	val := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	n := 0
	for _, c := range val {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
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
