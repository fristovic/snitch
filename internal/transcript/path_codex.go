package transcript

// CodexPathResolver resolves project/session metadata from Codex rollout paths:
//
//	~/.codex/sessions/YYYY/MM/DD/rollout-<ts>-<uuid>.jsonl
//
// Codex does not encode the project cwd in the path — it lives in the
// session_meta payload's "cwd" field. The watcher applies ParsedLine.Cwd from
// the parser when ingesting rollout files.
type CodexPathResolver struct{}

func (CodexPathResolver) ProjectCwd(path string) string { return "" }

func (CodexPathResolver) ProjectDir(path string) string { return "" }

func (CodexPathResolver) SessionID(path string) string {
	return sessionIDFromFilename(path, "rollout-")
}
