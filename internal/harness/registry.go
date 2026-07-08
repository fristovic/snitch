// Package harness holds the per-harness descriptor registry.
//
// A Descriptor bundles everything the daemon and verify pipeline need to serve
// one agent platform (Cursor, Claude Code, Codex, Pi, OpenCode):
//
//   - ingestion: an Ingest closure that starts the harness's source
//     (fsnotify watcher for JSONL harnesses, SQLite poll reader for OpenCode)
//   - shell output: a ShellOutputResolver (Cursor terminal files, or noop)
//
// The Registry is keyed by harness name ("cursor", "claude", ...). The daemon
// iterates enabled harnesses to start ingestion; the verify layer looks up the
// shell-output resolver for each turn's harness.
package harness

import (
	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/transcript"
)

// Stopper is anything the daemon must stop on shutdown.
type Stopper interface {
	Stop() error
}

// Descriptor bundles everything needed to ingest and verify one harness.
type Descriptor struct {
	Name  string
	Shell transcript.ShellOutputResolver
	// Ingest starts the harness's transcript source rooted at root (watch
	// directory for JSONL harnesses, DB path for OpenCode) and returns its
	// stopper.
	Ingest func(bus *event.Bus, root string) (Stopper, error)
}

// jsonlIngest builds an Ingest closure for an fsnotify-watched JSONL harness.
func jsonlIngest(name string, parser transcript.TranscriptParser, resolver transcript.PathResolver, ownsFile, ownsDir func(string) bool) func(*event.Bus, string) (Stopper, error) {
	return func(bus *event.Bus, root string) (Stopper, error) {
		w := transcript.NewWatcherWith(bus, transcript.WatcherConfig{
			Harness:  name,
			Root:     root,
			Parser:   parser,
			Resolver: resolver,
			OwnsFile: ownsFile,
			OwnsDir:  ownsDir,
			Enabled:  true,
		})
		if err := w.Start(); err != nil {
			return nil, err
		}
		return w, nil
	}
}

// Registry maps harness name → Descriptor.
type Registry struct {
	m map[string]Descriptor
}

// NewRegistry builds a registry pre-populated with all built-in harnesses.
func NewRegistry() *Registry {
	r := &Registry{m: make(map[string]Descriptor)}
	for _, d := range []Descriptor{
		cursorDescriptor(),
		claudeDescriptor(),
		codexDescriptor(),
		piDescriptor(),
		opencodeDescriptor(),
	} {
		if d.Shell == nil {
			d.Shell = transcript.NoopShellOutputResolver()
		}
		r.m[d.Name] = d
	}
	return r
}

// All returns every registered descriptor (order not guaranteed).
func (r *Registry) All() []Descriptor {
	out := make([]Descriptor, 0, len(r.m))
	for _, d := range r.m {
		out = append(out, d)
	}
	return out
}

// ShellResolver returns the shell-output resolver for a harness, or nil for
// unknown/empty harness names (callers should treat nil as "no artifact
// resolution; inline tool results only").
func (r *Registry) ShellResolver(name string) transcript.ShellOutputResolver {
	d, ok := r.m[name]
	if !ok {
		return nil
	}
	return d.Shell
}
