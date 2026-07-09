package event

import "github.com/fristovic/snitch/internal/record"

// TopFalseClaim is the highest-severity false claim on a verified run.
// Used by Snitch Bar notifications and other run.completed consumers.
type TopFalseClaim struct {
	ClaimType     string `json:"claim_type,omitempty"`
	Source        string `json:"source,omitempty"`
	Target        string `json:"target,omitempty"`
	Claimed       string `json:"claimed,omitempty"`
	Actual        string `json:"actual,omitempty"`
	ClaimSentence string `json:"claim_sentence,omitempty"`
	ClaimContext  string `json:"claim_context,omitempty"`
}

// TopFalseClaimFromRecord builds TopFalseClaim from a persisted claim.
func TopFalseClaimFromRecord(c record.Claim) TopFalseClaim {
	return TopFalseClaim{
		ClaimType:     c.ClaimType,
		Source:        c.Source,
		Target:        c.Target,
		Claimed:       c.Claimed,
		Actual:        c.Actual,
		ClaimSentence: c.ClaimSentence,
		ClaimContext:  c.ClaimContext,
	}
}

// RunVerifiedPayload is published when verification completes.
type RunVerifiedPayload struct {
	RunID       string         `json:"run_id"`
	Verdict     record.Verdict `json:"verdict"`
	MaxSeverity int            `json:"max_severity"`
	Command     string         `json:"command,omitempty"`
	ProjectPath string         `json:"project_path,omitempty"`
	SessionID   string         `json:"session_id,omitempty"`
	// TopClaim is set when the run has at least one false claim
	// (record.SelectTopFalseClaim policy: verified=-1, max severity).
	TopClaim *TopFalseClaim `json:"top_claim,omitempty"`

	// Flat aliases kept for one release so ≤0.3.x readers keep working.
	// Prefer TopClaim. Removed in a later minor.
	TopClaimType     string `json:"top_claim_type,omitempty"`
	TopClaimed       string `json:"top_claimed,omitempty"`
	TopActual        string `json:"top_actual,omitempty"`
	TopClaimSentence string `json:"top_claim_sentence,omitempty"`
	TopClaimContext  string `json:"top_claim_context,omitempty"`
}
