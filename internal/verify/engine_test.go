package verify_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/fristovic/snitch/internal/capture"
	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/transcript"
	"github.com/fristovic/snitch/internal/verify"
)

func TestEngineVerifiesToolCalls(t *testing.T) {
	dir := t.TempDir()
	store, _ := record.Open(dir)
	defer store.Close()
	deviceID, _ := store.EnsureDeviceID()

	bus := event.NewBus()
	defer bus.Close()

	done := make(chan struct{}, 1)
	engine := verify.NewEngine(bus, store, config.Default().Verification, deviceID, nil)
	engine.OnVerified(func(event.RunVerifiedPayload) { done <- struct{}{} })
	engine.Start()

	payload := capture.RunPayload{
		RunID:         "run-1",
		SessionID:     "s1",
		ProjectPath:   t.TempDir(),
		Output:        "user\nassistant",
		ToolCallCount: 1,
		ToolCalls: []transcript.ToolCall{
			{Name: "Read", Target: "missing.go"},
		},
		StartedAt:  time.Now().Add(-time.Minute),
		FinishedAt: time.Now(),
	}
	data, _ := json.Marshal(payload)
	bus.Publish(event.Event{Type: event.EventRunCaptured, Payload: data, ID: "run-1", Timestamp: time.Now()})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}

	claims, err := store.GetClaimsByRun("run-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(claims) != 1 {
		t.Fatalf("claims: %+v", claims)
	}
	if claims[0].ClaimType != "tool_read" {
		t.Fatalf("unexpected claim: %+v", claims[0])
	}
}

func TestEngineTestPassNoEvidenceIsMissingNotFail(t *testing.T) {
	dir := t.TempDir()
	store, _ := record.Open(dir)
	defer store.Close()
	deviceID, _ := store.EnsureDeviceID()

	bus := event.NewBus()
	defer bus.Close()

	done := make(chan struct{}, 1)
	engine := verify.NewEngine(bus, store, config.Default().Verification, deviceID, nil)
	engine.OnVerified(func(p event.RunVerifiedPayload) {
		done <- struct{}{}
	})
	engine.Start()

	payload := capture.RunPayload{
		RunID:         "run-missing-test",
		ProjectPath:   t.TempDir(),
		AssistantText: "All tests pass and everything looks good.",
		Output:        "All tests pass",
		StartedAt:     time.Now().Add(-time.Minute),
		FinishedAt:    time.Now(),
	}
	data, _ := json.Marshal(payload)
	bus.Publish(event.Event{Type: event.EventRunCaptured, Payload: data, ID: "run-missing-test", Timestamp: time.Now()})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for verified run")
	}

	run, _ := store.GetRunByID("run-missing-test")
	if run == nil || run.Verdict != record.VerdictPass {
		t.Fatalf("expected pass verdict (missing only), got %+v", run)
	}

	claims, _ := store.GetClaimsByRun("run-missing-test")
	var testPass *record.Claim
	for i := range claims {
		if claims[i].ClaimType == "test_pass" {
			testPass = &claims[i]
			break
		}
	}
	if testPass == nil {
		t.Fatalf("expected test_pass claim, got %+v", claims)
	}
	if testPass.Epistemic != "missing" {
		t.Fatalf("expected missing epistemic, got %+v", testPass)
	}
	if record.IsContradictedClaim(*testPass) {
		t.Fatal("missing claim must not count as contradicted")
	}
}

func TestEngineCatchesTestPassContradicted(t *testing.T) {
	dir := t.TempDir()
	store, _ := record.Open(dir)
	defer store.Close()
	deviceID, _ := store.EnsureDeviceID()

	bus := event.NewBus()
	defer bus.Close()

	done := make(chan struct{}, 1)
	engine := verify.NewEngine(bus, store, config.Default().Verification, deviceID, nil)
	engine.OnVerified(func(p event.RunVerifiedPayload) {
		if p.Verdict == record.VerdictFail {
			done <- struct{}{}
		}
	})
	engine.Start()

	payload := capture.RunPayload{
		RunID:         "run-false-claim",
		ProjectPath:   t.TempDir(),
		AssistantText: "All tests pass and everything looks good.",
		Output:        "All tests pass",
		ToolCalls: []transcript.ToolCall{{
			Name:    "Shell",
			Target:  "go test ./...",
			Result:  "FAIL\tpkg\n",
			IsError: true,
		}},
		StartedAt:  time.Now().Add(-time.Minute),
		FinishedAt: time.Now(),
	}
	data, _ := json.Marshal(payload)
	bus.Publish(event.Event{Type: event.EventRunCaptured, Payload: data, ID: "run-false-claim", Timestamp: time.Now()})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for fail verdict")
	}

	claims, _ := store.GetClaimsByRun("run-false-claim")
	found := false
	for _, c := range claims {
		if c.ClaimType == "test_pass" && record.IsContradictedClaim(c) {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected contradicted test_pass, got %+v", claims)
	}
}
