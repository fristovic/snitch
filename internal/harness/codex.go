package harness

import (
	"path/filepath"
	"strings"

	"github.com/fristovic/snitch/internal/transcript"
)

// codexDescriptor describes Codex CLI's transcript ingestion.
// Codex stores sessions under ~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl.
// Shell output is inline in function_call_output, so the resolver is noop.
func codexDescriptor() Descriptor {
	return Descriptor{
		Name: "codex",
		Ingest: jsonlIngest("codex", transcript.CodexParser{}, transcript.CodexPathResolver{},
			func(path string) bool {
				return strings.HasSuffix(path, ".jsonl") && strings.HasPrefix(filepath.Base(path), "rollout-")
			},
			func(path string) bool {
				return strings.Contains(path, "/.codex/sessions")
			}),
	}
}
