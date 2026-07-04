package capture

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/scrub"
	"github.com/fristovic/snitch/internal/transcript"
)

const maxOutputBytes = 10 * 1024 * 1024

// RunPayload is emitted on RunCaptured.
type RunPayload struct {
	RunID          string                `json:"run_id"`
	SessionID      string                `json:"session_id"`
	TranscriptPath string                `json:"transcript_path"`
	ProjectPath    string                `json:"project_path"`
	StartHEAD      string                `json:"start_head,omitempty"`
	Output         string                `json:"output"`
	UserText       string                `json:"user_text"`
	AssistantText  string                `json:"assistant_text"`
	ToolCalls      []transcript.ToolCall `json:"tool_calls"`
	ToolCallCount  int                   `json:"tool_call_count"`
	Harness        string                `json:"harness,omitempty"`
	Command        string                `json:"command,omitempty"`
	StartedAt      time.Time             `json:"started_at"`
	FinishedAt     time.Time             `json:"finished_at"`
	EndHEAD        string                `json:"end_head,omitempty"`
	FileManifest   map[string]string     `json:"file_manifest,omitempty"`
}

// Engine builds RunCaptured events from completed Cursor turns.
type Engine struct {
	bus    *event.Bus
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a capture engine.
func New(bus *event.Bus) *Engine {
	return &Engine{bus: bus}
}

// Start listens for turn completion events.
func (e *Engine) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	e.ctx = ctx
	e.cancel = cancel

	turnCh := e.bus.Subscribe(event.EventTurnCompleted)
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		for {
			select {
			case <-e.ctx.Done():
				return
			case ev, ok := <-turnCh:
				if !ok {
					return
				}
				e.onTurnCompleted(ev)
			}
		}
	}()
}

func (e *Engine) onTurnCompleted(ev event.Event) {
	var turn transcript.TurnCompleted
	if err := json.Unmarshal(ev.Payload, &turn); err != nil {
		slog.Warn("invalid TurnCompleted payload", "err", err)
		return
	}

	output := scrub.Scrub(strings.TrimSpace(turn.UserText + "\n" + turn.AssistantText))
	if len(output) > maxOutputBytes {
		output = output[:maxOutputBytes]
	}

	command := truncate(turn.UserText, 500)
	payload := RunPayload{
		RunID:          turn.RunID,
		SessionID:      turn.SessionID,
		TranscriptPath: turn.TranscriptPath,
		ProjectPath:    turn.ProjectPath,
		StartHEAD:      turn.StartHEAD,
		Output:         output,
		UserText:       turn.UserText,
		AssistantText:  turn.AssistantText,
		ToolCalls:      turn.ToolCalls,
		ToolCallCount:  len(turn.ToolCalls),
		Harness:        "cursor",
		Command:        command,
		StartedAt:      turn.StartedAt,
		FinishedAt:     turn.FinishedAt,
		EndHEAD:        turn.EndHEAD,
		FileManifest:   turn.FileManifest,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("marshal RunCaptured", "err", err)
		return
	}
	e.bus.Publish(event.Event{
		ID: turn.RunID, Timestamp: turn.FinishedAt, Source: "capture",
		Type: event.EventRunCaptured, Payload: data,
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// Stop waits for goroutines.
func (e *Engine) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
	e.wg.Wait()
}

// HashOutput returns SHA256 hex of output.
func HashOutput(output string) string {
	h := sha256.Sum256([]byte(output))
	return hex.EncodeToString(h[:])
}

// NowUTC returns current UTC time.
func NowUTC() time.Time {
	return time.Now().UTC()
}
