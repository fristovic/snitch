package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/transcript"
)

// TestWatcherLiveIngestion exercises the full fsnotify path per JSONL harness:
// a transcript file grows line by line (as a real agent session does) and the
// watcher must assemble and publish correct TurnCompleted events, including
// the idle-flush path for harnesses with no trailing end marker.
func TestWatcherLiveIngestion(t *testing.T) {
	cases := []struct {
		harness  string
		parser   transcript.TranscriptParser
		fixture  string
		relPath  string // transcript path relative to the watch root
		ownsFile func(string) bool
		ownsDir  func(string) bool
		// idleOnly means the fixture has no explicit trailing end marker, so
		// the final (or only) turn arrives via idle-flush.
		idleOnly  bool
		wantTurns int
		wantTool  string
	}{
		{
			harness: "cursor",
			parser:  transcript.CursorParser{},
			fixture: "cursor_tools.jsonl",
			relPath: "proj/agent-transcripts/sess-1/sess-1.jsonl",
			ownsFile: func(p string) bool {
				return strings.Contains(p, "agent-transcripts") && strings.HasSuffix(p, ".jsonl")
			},
			ownsDir:   func(p string) bool { return strings.Contains(p, "agent-transcripts") },
			wantTurns: 1,
			wantTool:  "Write",
		},
		{
			harness:   "claude",
			parser:    transcript.ClaudeParser{},
			fixture:   "claude_assistant.jsonl",
			relPath:   "projects/-Users-alice-app/sess-1.jsonl",
			ownsFile:  func(p string) bool { return strings.HasSuffix(p, ".jsonl") },
			ownsDir:   func(p string) bool { return true },
			wantTurns: 1,
			wantTool:  "Shell",
		},
		{
			harness:   "codex",
			parser:    transcript.CodexParser{},
			fixture:   "codex_rollout.jsonl",
			relPath:   "sessions/2026/07/08/rollout-1-abc.jsonl",
			ownsFile:  func(p string) bool { return strings.HasPrefix(filepath.Base(p), "rollout-") },
			ownsDir:   func(p string) bool { return true },
			wantTurns: 1,
			wantTool:  "Shell",
		},
		{
			harness:   "pi",
			parser:    transcript.PiParser{},
			fixture:   "pi_session.jsonl",
			relPath:   "sessions/--Users-alice-app--/2026_sess.jsonl",
			ownsFile:  func(p string) bool { return strings.HasSuffix(p, ".jsonl") },
			ownsDir:   func(p string) bool { return true },
			idleOnly:  true, // Pi's final turn only flushes via idle timer
			wantTurns: 1,
			wantTool:  "Shell",
		},
	}

	for _, tc := range cases {
		t.Run(tc.harness, func(t *testing.T) {
			root := t.TempDir()
			bus := event.NewBus()
			defer bus.Close()
			turnCh := bus.Subscribe(event.EventTurnCompleted)

			w := transcript.NewWatcherWith(bus, transcript.WatcherConfig{
				Harness:   tc.harness,
				Root:      root,
				Parser:    tc.parser,
				Resolver:  resolverFor(tc.harness),
				OwnsFile:  tc.ownsFile,
				OwnsDir:   tc.ownsDir,
				Enabled:   true,
				IdleFlush: 300 * time.Millisecond, // fast idle-flush for the test
			})
			if err := w.Start(); err != nil {
				t.Fatal(err)
			}
			defer w.Stop()

			// Create the session directory AFTER the watcher started (the
			// real-world case: a new session appears mid-run).
			path := filepath.Join(root, filepath.FromSlash(tc.relPath))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			// Give fsnotify a beat to register the new directories.
			time.Sleep(200 * time.Millisecond)

			// Append fixture lines one at a time, like a live session.
			fixture, err := os.ReadFile(filepath.Join("..", "fixtures", "sample_transcripts", tc.fixture))
			if err != nil {
				t.Fatal(err)
			}
			f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				t.Fatal(err)
			}
			for _, line := range strings.Split(strings.TrimSpace(string(fixture)), "\n") {
				if _, err := f.WriteString(line + "\n"); err != nil {
					t.Fatal(err)
				}
				_ = f.Sync()
				time.Sleep(30 * time.Millisecond)
			}
			f.Close()

			deadline := time.After(5 * time.Second)
			var turns []transcript.TurnCompleted
			for len(turns) < tc.wantTurns {
				select {
				case ev := <-turnCh:
					var turn transcript.TurnCompleted
					if err := json.Unmarshal(ev.Payload, &turn); err != nil {
						t.Fatal(err)
					}
					turns = append(turns, turn)
				case <-deadline:
					t.Fatalf("timed out: got %d turns, want %d (idleOnly=%v)", len(turns), tc.wantTurns, tc.idleOnly)
				}
			}

			var sawTool bool
			for _, turn := range turns {
				if turn.Harness != tc.harness {
					t.Errorf("harness=%q want %q", turn.Harness, tc.harness)
				}
				for _, call := range turn.ToolCalls {
					if call.Name == tc.wantTool {
						sawTool = true
					}
				}
			}
			if !sawTool {
				t.Errorf("no %q tool call in turns: %+v", tc.wantTool, turns)
			}
		})
	}
}

// resolverFor returns the production path resolver for a harness.
func resolverFor(harness string) transcript.PathResolver {
	switch harness {
	case "cursor":
		return transcript.CursorPathResolver{}
	case "claude":
		return transcript.ClaudePathResolver{}
	case "codex":
		return transcript.CodexPathResolver{}
	case "pi":
		return transcript.PiPathResolver{}
	default:
		return transcript.OpenCodePathResolver{}
	}
}
