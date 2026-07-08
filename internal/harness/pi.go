package harness

import (
	"strings"

	"github.com/fristovic/snitch/internal/transcript"
)

// piDescriptor describes Pi's transcript ingestion.
// Pi stores sessions under ~/.pi/agent/sessions/--<cwd>--/*.jsonl.
// Shell output is inline (bashExecution messages carry command/output/exitCode),
// so the shell resolver is noop and the inline fallback handles output.
func piDescriptor() Descriptor {
	return Descriptor{
		Name: "pi",
		Ingest: jsonlIngest("pi", transcript.PiParser{}, transcript.PiPathResolver{},
			func(path string) bool {
				return strings.Contains(path, "/.pi/agent/sessions/") && strings.HasSuffix(path, ".jsonl")
			},
			func(path string) bool {
				return strings.Contains(path, "/.pi/agent/sessions")
			}),
	}
}
