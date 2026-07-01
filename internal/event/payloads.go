package event

import "github.com/fristovic/snitch/internal/record"

// RunVerifiedPayload is published when verification completes.
type RunVerifiedPayload struct {
	RunID       string         `json:"run_id"`
	Verdict     record.Verdict `json:"verdict"`
	MaxSeverity int            `json:"max_severity"`
	Command     string         `json:"command,omitempty"`
	ProjectPath string         `json:"project_path,omitempty"`
	SessionID   string         `json:"session_id,omitempty"`
}
