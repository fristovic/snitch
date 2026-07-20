package transcript

import (
	"bufio"
	"log/slog"
	"os"
)

// CodexPathResolver resolves project/session metadata from Codex rollout paths:
//
//	~/.codex/sessions/YYYY/MM/DD/rollout-<ts>-<uuid>.jsonl
//
// Codex does not encode the project cwd in the path — it lives in the
// session_meta payload's "cwd" field. The watcher applies ParsedLine.Cwd from
// the parser when ingesting rollout files.
type CodexPathResolver struct{}

func (CodexPathResolver) ProjectCwd(path string) string {
	cwd := codexCwdFromSessionMeta(path)
	if cwd == "" {
		slog.Warn("codex project cwd unknown; session_meta missing or unparseable", "path", path)
	}
	return cwd
}

func codexCwdFromSessionMeta(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	parser := CodexParser{}
	scanner := bufio.NewScanner(f)
	for i := 0; i < 20 && scanner.Scan(); i++ {
		pl, ok := parser.ParseLine(scanner.Text())
		if ok && pl.Cwd != "" {
			return pl.Cwd
		}
	}
	return ""
}

func (CodexPathResolver) ProjectDir(path string) string { return "" }

func (CodexPathResolver) SessionID(path string) string {
	return sessionIDFromFilename(path, "rollout-")
}
