package record_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/fristovic/snitch/internal/capture"
	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/transcript"
)

func TestTurnSnapshotRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := record.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	run := record.Run{
		ID:        "run-snap-1",
		SessionID: "sess-snap",
		Verdict:   record.VerdictPass,
	}
	if err := store.InsertRun(run); err != nil {
		t.Fatal(err)
	}

	payload := capture.RunPayload{
		RunID:         "run-snap-1",
		SessionID:     "sess-snap",
		AssistantText: "done",
		ToolCalls:     []transcript.ToolCall{{Name: "Read", Target: "main.go"}},
		StartedAt:     time.Now().Add(-time.Minute),
		FinishedAt:    time.Now(),
		StartHEAD:     "aaa",
		EndHEAD:       "bbb",
		FileManifest:  map[string]string{"main.go": "deadbeef"},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTurnSnapshot("run-snap-1", raw, "aaa", "bbb", payload.FileManifest); err != nil {
		t.Fatal(err)
	}

	got, err := store.GetRunPayloadJSON("run-snap-1")
	if err != nil || len(got) == 0 {
		t.Fatalf("payload json: err=%v len=%d", err, len(got))
	}
	var decoded capture.RunPayload
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.EndHEAD != "bbb" || len(decoded.ToolCalls) != 1 {
		t.Fatalf("decoded: %+v", decoded)
	}

	priors, err := store.GetPriorRunPayloadJSON("sess-snap", time.Now().Add(time.Hour), 3)
	if err != nil || len(priors) != 1 {
		t.Fatalf("priors: err=%v len=%d", err, len(priors))
	}
}
