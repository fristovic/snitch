package integration_test

import (
	"testing"
	"time"

	"github.com/fristovic/snitch/internal/capture"
	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/transcript"
	"github.com/fristovic/snitch/internal/verify"
)

func TestSessionLookbackThreeTurns(t *testing.T) {
	dir := t.TempDir()
	projectDir := t.TempDir()
	store, err := record.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	deviceID, _ := store.EnsureDeviceID()

	sessionID := "sess-lookback"
	base := time.Now().Add(-time.Hour)

	turns := []capture.RunPayload{
		{
			RunID: "run-lb-1", SessionID: sessionID, ProjectPath: projectDir,
			AssistantText: "committing", ToolCalls: []transcript.ToolCall{
				{Name: "Shell", Target: "git commit -m x"},
			},
			StartHEAD: "head1", EndHEAD: "head2",
			StartedAt: base, FinishedAt: base.Add(30 * time.Second),
		},
		{
			RunID: "run-lb-2", SessionID: sessionID, ProjectPath: projectDir,
			AssistantText: "reading",
			StartedAt:     base.Add(time.Minute), FinishedAt: base.Add(90 * time.Second),
		},
		{
			RunID: "run-lb-3", SessionID: sessionID, ProjectPath: projectDir,
			AssistantText: "### Summary\nI've committed the changes.",
			StartedAt:     base.Add(2 * time.Minute), FinishedAt: base.Add(150 * time.Second),
		},
	}

	bus := event.NewBus()
	defer bus.Close()
	engine := verify.NewEngine(bus, store, config.Default().Verification, deviceID)
	for _, p := range turns {
		engine.VerifyPayload(p)
	}

	claims, err := store.GetClaimsByRun("run-lb-3")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range claims {
		if c.ClaimType == "committed" && c.Verified < 0 && c.Severity >= 2 {
			t.Fatalf("expected commit credit from lookback, got lie: %+v", c)
		}
	}
}
