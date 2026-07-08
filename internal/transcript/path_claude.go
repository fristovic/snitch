package transcript

// ClaudePathResolver resolves project/session metadata from Claude Code
// transcript paths: ~/.claude/projects/<slug>/<session-uuid>.jsonl
//
// Claude uses slug encoding with a leading dash (-Users-a-b-c → /Users/a/b/c);
// slug decoding is shared with Cursor. Session files live directly under the
// project dir, so the session id is the filename stem.
type ClaudePathResolver struct{}

func (ClaudePathResolver) ProjectCwd(path string) string { return slugProjectCwd(path) }

func (ClaudePathResolver) ProjectDir(path string) string { return slugProjectDir(path, ".claude") }

func (ClaudePathResolver) SessionID(path string) string { return sessionIDFromFilename(path, "") }
