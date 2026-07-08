package harness

import (
	"strings"

	"github.com/fristovic/snitch/internal/transcript"
)

// claudeDescriptor describes Claude Code's transcript ingestion.
// Shell output is inline in tool_result (no terminal-file fallback), so the
// resolver is a no-op and the verifier's inline path handles it.
func claudeDescriptor() Descriptor {
	return Descriptor{
		Name: "claude",
		Ingest: jsonlIngest("claude", transcript.ClaudeParser{}, transcript.ClaudePathResolver{},
			func(path string) bool {
				return strings.Contains(path, ".claude/projects/") && strings.HasSuffix(path, ".jsonl")
			},
			func(path string) bool {
				return strings.Contains(path, ".claude/projects")
			}),
	}
}
