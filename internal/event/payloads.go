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
	// Top-lie fields are set when the run has at least one false claim (same
	// selection rules as notify.TopLieClaim).
	TopClaimType string `json:"top_claim_type,omitempty"`
	TopClaimed   string `json:"top_claimed,omitempty"`
	TopActual    string `json:"top_actual,omitempty"`
}
