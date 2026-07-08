package integration_test

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/fristovic/snitch/internal/capture"
	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/transcript"
	"github.com/fristovic/snitch/internal/verify"
)

// TestMultiHarnessPipelineParseVerifyStore runs each JSONL harness's fixture
// through the full parse→capture→verify→store pipeline and asserts the harness
// tag flows through to the stored run. OpenCode (SQLite) is covered by the
// reader unit test; this test covers the four fsnotify-watched harnesses.
func TestMultiHarnessPipelineParseVerifyStore(t *testing.T) {
	fixtures := []struct {
		harness string
		parser  transcript.TranscriptParser
		fixture string
	}{
		{"cursor", transcript.CursorParser{}, "cursor_tools.jsonl"},
		{"claude", transcript.ClaudeParser{}, "claude_assistant.jsonl"},
		{"codex", transcript.CodexParser{}, "codex_rollout.jsonl"},
		{"pi", transcript.PiParser{}, "pi_session.jsonl"},
	}

	for _, f := range fixtures {
		t.Run(f.harness, func(t *testing.T) {
			dir := t.TempDir()
			store, err := record.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			deviceID, _ := store.EnsureDeviceID()

			bus := event.NewBus()
			defer bus.Close()

			capEngine := capture.New(bus)
			capEngine.Start()
			defer capEngine.Stop()

			done := make(chan string, 1)
			verifyEngine := verify.NewEngine(bus, store, config.Default().Verification, deviceID, nil)
			verifyEngine.OnVerified(func(p event.RunVerifiedPayload) {
				select {
				case done <- p.RunID:
				default:
				}
			})
			verifyEngine.Start()

			// Parse the fixture and extract a representative tool call + text,
			// then synthesize a TurnCompleted carrying the harness tag.
			path := filepath.Join("..", "fixtures", "sample_transcripts", f.fixture)
			lines, _, err := transcript.ParseLinesWith(f.parser, path, 0)
			if err != nil {
				t.Fatal(err)
			}
			var allCalls []transcript.ToolCall
			var asstText string
			for _, l := range lines {
				allCalls = append(allCalls, l.ToolCalls...)
				if l.Role == "assistant" && l.Text != "" {
					asstText += l.Text + " "
				}
			}
			if len(allCalls) == 0 {
				t.Fatalf("%s: expected at least one tool call from fixture", f.harness)
			}

			runID := f.harness + "-run-1"
			turn := transcript.TurnCompleted{
				RunID:          runID,
				SessionID:      f.harness + "-sess",
				TranscriptPath: path,
				ProjectPath:    dir,
				Harness:        f.harness,
				UserText:       "do the task",
				AssistantText:  asstText,
				ToolCalls:      transcript.AttachToolResults(allCalls, nil),
				StartedAt:      time.Now().Add(-time.Minute),
				FinishedAt:     time.Now(),
			}
			payload, _ := json.Marshal(turn)
			bus.Publish(event.Event{
				ID: runID, Timestamp: turn.FinishedAt, Source: "test",
				Type: event.EventTurnCompleted, Payload: payload,
			})

			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatalf("%s: run not verified in time", f.harness)
			}

			run, err := store.GetRunByID(runID)
			if err != nil || run == nil {
				t.Fatalf("%s: expected run in store", f.harness)
			}
			if run.Harness != f.harness {
				t.Errorf("%s: harness mismatch got=%q want=%q", f.harness, run.Harness, f.harness)
			}
			// Each harness should normalize to at least one canonical tool name.
			claims, _ := store.GetClaimsByRun(run.ID)
			if len(claims) == 0 {
				t.Errorf("%s: expected at least one claim", f.harness)
			}
		})
	}
}
