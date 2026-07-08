package harness

import (
	"strings"

	"github.com/fristovic/snitch/internal/transcript"
)

// cursorDescriptor describes Cursor's transcript ingestion.
// Tool names are already canonical (Shell, Write, StrReplace, ...). Shell
// output resolves from per-project terminals/*.txt dump files.
func cursorDescriptor() Descriptor {
	return Descriptor{
		Name:  "cursor",
		Shell: transcript.CursorTerminalResolver{},
		Ingest: jsonlIngest("cursor", transcript.CursorParser{}, transcript.CursorPathResolver{},
			func(path string) bool {
				return strings.Contains(path, "agent-transcripts") && strings.HasSuffix(path, ".jsonl")
			},
			func(path string) bool {
				return strings.Contains(path, "agent-transcripts")
			}),
	}
}
