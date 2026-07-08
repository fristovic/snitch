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

func TestTranscriptParseFixture(t *testing.T) {
	path := filepath.Join("..", "fixtures", "sample_transcripts", "cursor_tools.jsonl")
	lines, _, err := transcript.ParseLinesWith(transcript.CursorParser{}, path, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) < 2 {
		t.Fatalf("expected lines, got %d", len(lines))
	}
}

func TestPipelineParseVerifyStore(t *testing.T) {
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

	verified := make(chan record.Verdict, 1)
	verifyEngine := verify.NewEngine(bus, store, config.Default().Verification, deviceID, nil)
	verifyEngine.OnVerified(func(p event.RunVerifiedPayload) {
		select {
		case verified <- p.Verdict:
		default:
		}
	})
	verifyEngine.Start()

	projectDir := t.TempDir()
	path := filepath.Join("..", "fixtures", "sample_transcripts", "cursor_tools.jsonl")
	turn := transcript.TurnCompleted{
		RunID:          "run-test-1",
		SessionID:      "sess-1",
		TranscriptPath: path,
		ProjectPath:    projectDir,
		UserText:       "fix bug",
		AssistantText:  "done",
		ToolCalls: []transcript.ToolCall{
			{Name: "Read", Target: "main.go"},
		},
		StartedAt:  time.Now().Add(-time.Minute),
		FinishedAt: time.Now(),
	}
	payload, _ := json.Marshal(turn)
	bus.Publish(event.Event{
		ID: turn.RunID, Timestamp: turn.FinishedAt, Source: "test",
		Type: event.EventTurnCompleted, Payload: payload,
	})

	select {
	case <-verified:
	case <-time.After(3 * time.Second):
		run, _ := store.GetRunByID("run-test-1")
		if run == nil {
			t.Fatal("run was not inserted")
		}
		t.Fatalf("run not verified, verdict=%s", run.Verdict)
	}

	run, err := store.GetRunByID("run-test-1")
	if err != nil || run == nil {
		t.Fatal("expected run in store")
	}
	if run.SessionID != "sess-1" {
		t.Fatalf("session id: %s", run.SessionID)
	}
	claims, _ := store.GetClaimsByRun(run.ID)
	if len(claims) == 0 {
		t.Fatal("expected claims")
	}
}

func TestPipelineTestPassLieEndToEnd(t *testing.T) {
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

	verifyEngine := verify.NewEngine(bus, store, config.Default().Verification, deviceID, nil)
	verifyEngine.Start()

	turn := transcript.TurnCompleted{
		RunID:         "run-lie-e2e",
		ProjectPath:   t.TempDir(),
		UserText:      "run tests",
		AssistantText: "All tests pass. You're good to go.",
		StartedAt:     time.Now().Add(-time.Minute),
		FinishedAt:    time.Now(),
	}
	payload, _ := json.Marshal(turn)
	bus.Publish(event.Event{
		ID: turn.RunID, Timestamp: turn.FinishedAt, Source: "test",
		Type: event.EventTurnCompleted, Payload: payload,
	})

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		run, _ := store.GetRunByID("run-lie-e2e")
		if run != nil && run.Verdict == record.VerdictFail {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	run, _ := store.GetRunByID("run-lie-e2e")
	if run == nil || run.Verdict != record.VerdictFail {
		t.Fatalf("expected fail verdict, got %+v", run)
	}

	claims, _ := store.GetClaims(record.ClaimFilter{LiesOnly: true, ClaimType: "test_pass"})
	if len(claims) == 0 {
		t.Fatal("expected test_pass lie via get_claims filter")
	}
}
