package transcript

import (
	"encoding/json"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/fristovic/snitch/internal/event"
)

// PublishTurnCompleted marshals a completed turn and publishes it on the bus.
// Shared by the JSONL watcher and the OpenCode reader so the event shape has
// exactly one producer path.
func PublishTurnCompleted(bus *event.Bus, t TurnCompleted) {
	data, err := json.Marshal(t)
	if err != nil {
		slog.Warn("marshal TurnCompleted", "err", err)
		return
	}
	bus.Publish(event.Event{
		ID:        t.RunID,
		Timestamp: t.FinishedAt,
		Source:    "transcript",
		Type:      event.EventTurnCompleted,
		Payload:   data,
	})
	slog.Info("turn completed",
		"run_id", t.RunID,
		"harness", t.Harness,
		"session_id", t.SessionID,
		"project", t.ProjectPath,
		"tool_calls", len(t.ToolCalls),
	)
}

// GitHEAD returns the current HEAD commit of the git repo at dir, or "" if
// dir is empty or not a repo. Shared by turn ingestion (start/end HEAD
// snapshots) and the verify layer.
func GitHEAD(dir string) string {
	if dir == "" {
		return ""
	}
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
