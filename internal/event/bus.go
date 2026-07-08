package event

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

// EventType identifies the kind of event on the bus.
type EventType string

const (
	EventTurnCompleted EventType = "TurnCompleted"
	EventRunCaptured   EventType = "RunCaptured"
)

// Event is a typed message on the internal pub/sub bus.
type Event struct {
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	Source    string          `json:"source"`
	Type      EventType       `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// Bus is an in-memory pub/sub event bus.
type Bus struct {
	mu          sync.RWMutex
	subscribers map[EventType][]chan Event
	closed      bool
}

// NewBus creates a new event bus.
func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[EventType][]chan Event),
	}
}

// Subscribe returns a channel that receives events of the given type.
func (b *Bus) Subscribe(t EventType) <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Event, 64)
	b.subscribers[t] = append(b.subscribers[t], ch)
	return ch
}

// Publish sends an event to all subscribers of its type. Events are dropped
// (with a warning) when a subscriber's buffer is full rather than blocking
// the producer.
func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return
	}
	for _, ch := range b.subscribers[e.Type] {
		select {
		case ch <- e:
		default:
			slog.Warn("event bus dropped event (subscriber buffer full)", "type", e.Type, "id", e.ID)
		}
	}
}

// Close shuts down the bus and closes all subscriber channels.
func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for _, subs := range b.subscribers {
		for _, ch := range subs {
			close(ch)
		}
	}
	b.subscribers = nil
}
