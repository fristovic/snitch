package harness

import (
	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/transcript"
)

// opencodeDescriptor describes OpenCode's SQLite-backed ingestion.
// The reader polls opencode.db for new turns (root = DB path). Shell output
// is inline in tool parts, so the shell resolver is noop.
func opencodeDescriptor() Descriptor {
	return Descriptor{
		Name: "opencode",
		Ingest: func(bus *event.Bus, root string) (Stopper, error) {
			reader := transcript.NewOpenCodeReader(root, bus)
			if err := reader.Start(); err != nil {
				return nil, err
			}
			return reader, nil
		},
	}
}
