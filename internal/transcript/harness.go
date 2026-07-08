package transcript

import "time"

// TranscriptParser parses one JSONL line into a normalized ParsedLine.
// Each JSONL-based harness (Cursor, Claude Code, Codex, Pi) implements this
// to translate its wire format into Snitch's internal vocabulary.
// Returns ok=false to skip a line (malformed, empty, or harness-specific junk).
type TranscriptParser interface {
	// ParseLine decodes a single JSONL record. ok=false means drop the line.
	ParseLine(line string) (pl ParsedLine, ok bool)
	// Harness is the harness identifier this parser serves ("cursor", "claude", ...).
	Harness() string
}

// PathResolver derives project/session metadata from a transcript path.
// Each harness encodes the project cwd and session id differently (or not at
// all in the path — OpenCode reads them from the DB), so resolution is
// per-harness.
type PathResolver interface {
	// ProjectCwd returns the working directory the agent ran in, or "" if unknown.
	ProjectCwd(path string) string
	// ProjectDir returns the harness's on-disk project directory (the parent of
	// transcript artifacts), or "" if N/A. Cursor uses this to locate terminals/.
	ProjectDir(path string) string
	// SessionID returns the session identifier encoded in path, or "".
	SessionID(path string) string
}

// WatcherConfig configures one fsnotify-based transcript watcher instance.
// The daemon builds one per enabled JSONL harness.
type WatcherConfig struct {
	Harness  string
	Root     string // watch root, e.g. ~/.cursor/projects
	Parser   TranscriptParser
	Resolver PathResolver
	// OwnsFile reports whether a file path is a transcript this harness owns.
	OwnsFile func(path string) bool
	// OwnsDir reports whether a directory should be watched (recurse into it).
	// Cursor watches dirs containing "agent-transcripts"; flat-layout harnesses
	// watch every dir under their root.
	OwnsDir func(path string) bool
	Enabled bool
	// IdleFlush is how long a turn buffer may sit without new writes before
	// the watcher flushes it (covers harnesses whose final turn has no end
	// marker). Zero means the default (30s).
	IdleFlush time.Duration
}

// ShellOutputRequest carries the context a harness needs to resolve shell
// output from its own on-disk artifacts (Cursor terminal files, etc.).
type ShellOutputRequest struct {
	TranscriptPath string
	ProjectPath    string
	Cwd            string
	Command        string
	StartedAt      time.Time
	FinishedAt     time.Time
}

// ShellOutputResolver resolves captured shell output + exit code for a tool
// call from harness-specific artifacts. Returning found=false lets the caller
// fall back to the inline tool_result. Each harness provides its own
// implementation; harnesses that embed output inline (Claude, Codex, Pi) use a
// no-op resolver and rely on the inline fallback.
type ShellOutputResolver interface {
	Resolve(tc ToolCall, req ShellOutputRequest) (output string, exitCode int, found bool)
}

// noopShellOutputResolver resolves nothing — used when a harness carries shell
// output inline in tool_result (Claude Code, Codex, Pi). The verifier's inline
// fallback handles those cases.
type noopShellOutputResolver struct{}

func (noopShellOutputResolver) Resolve(tc ToolCall, _ ShellOutputRequest) (string, int, bool) {
	return "", 0, false
}

// NoopShellOutputResolver is the shared no-op resolver for harnesses whose
// shell output is embedded inline in tool results.
func NoopShellOutputResolver() ShellOutputResolver { return noopShellOutputResolver{} }
