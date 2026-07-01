package capture_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/fristovic/snitch/internal/capture"
	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/transcript"
)

func TestTurnCompletedToRunPayloadRoundTrip(t *testing.T) {
	turn := transcript.TurnCompleted{
		RunID: "r1",
		ToolCalls: []transcript.ToolCall{
			{Name: "Read", Target: "main.go"},
		},
		FinishedAt: time.Now(),
	}
	data, _ := json.Marshal(turn)
	var decoded transcript.TurnCompleted
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.ToolCalls) != 1 {
		t.Fatalf("turn decode: %+v", decoded.ToolCalls)
	}

	bus := event.NewBus()
	defer bus.Close()
	cap := capture.New(bus)
	cap.Start()
	defer cap.Stop()

	ch := bus.Subscribe(event.EventRunCaptured)
	bus.Publish(event.Event{Type: event.EventTurnCompleted, Payload: data, Timestamp: time.Now()})

	select {
	case ev := <-ch:
		var payload capture.RunPayload
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if len(payload.ToolCalls) != 1 {
			t.Fatalf("payload tool calls: %+v", payload.ToolCalls)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}
