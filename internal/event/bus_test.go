package event_test

import (
	"testing"
	"time"

	"github.com/fristovic/snitch/internal/event"
)

func TestBusPublishSubscribe(t *testing.T) {
	bus := event.NewBus()
	defer bus.Close()
	ch := bus.Subscribe(event.EventTurnCompleted)
	bus.Publish(event.Event{
		ID:        "1",
		Timestamp: time.Now(),
		Source:    "test",
		Type:      event.EventTurnCompleted,
	})
	ev := <-ch
	if ev.ID != "1" {
		t.Fatalf("expected id 1, got %s", ev.ID)
	}
}

func TestBusWildcard(t *testing.T) {
	bus := event.NewBus()
	defer bus.Close()
	ch := bus.SubscribeAll()
	bus.Publish(event.Event{ID: "x", Type: event.EventRunCaptured, Timestamp: time.Now()})
	ev := <-ch
	if ev.Type != event.EventRunCaptured {
		t.Fatalf("unexpected type %s", ev.Type)
	}
}

func TestBusClose(t *testing.T) {
	bus := event.NewBus()
	ch := bus.Subscribe(event.EventRunVerified)
	bus.Close()
	_, ok := <-ch
	if ok {
		t.Fatal("channel should be closed")
	}
}
