package verifiers

import (
	"encoding/json"
	"time"

	"github.com/fristovic/snitch/internal/severity"
	"github.com/fristovic/snitch/internal/transcript"
)

// ClaimType identifies what the assistant claimed.
type ClaimType string

const (
	ClaimTestPass     ClaimType = "test_pass"
	ClaimCommitted    ClaimType = "committed"
	ClaimPushed       ClaimType = "pushed"
	ClaimFileCreated  ClaimType = "file_created"
	ClaimFileModified ClaimType = "file_modified"
	ClaimFileDeleted  ClaimType = "file_deleted"
	ClaimCommandRan   ClaimType = "command_ran"
	ClaimNoAction     ClaimType = "no_action"
)

// Claim is a prose or tool-derived claim to verify.
type Claim struct {
	Type        ClaimType      `json:"type"`
	Source      string         `json:"source"`
	Target      string         `json:"target"`
	Quote       string         `json:"quote"`
	Description string         `json:"description"`
	Input       map[string]any `json:"input,omitempty"`
}

// VerifyContext provides runtime context for verification.
type VerifyContext struct {
	Output         string
	Cwd            string
	ProjectPath    string
	StartHEAD      string
	TranscriptPath string
	ObservedAt     time.Time
	ToolCalls      []transcript.ToolCall
}

// Result is a verification outcome.
type Result struct {
	Claim       Claim
	Accurate    bool
	GroundTruth string
	Severity    severity.Level
	Evidence    []string
	Verifier    string
}

// Verifier checks claims against ground truth.
type Verifier interface {
	Name() string
	CanHandle(claim Claim) bool
	Verify(claim Claim, ctx VerifyContext) (Result, error)
}

// ToolCallToClaim maps a Cursor tool call to a verifiable claim.
func ToolCallToClaim(tc transcript.ToolCall) (Claim, bool) {
	switch tc.Name {
	case "Write", "StrReplace", "Delete", "Read", "Glob", "Shell", "Task":
		return Claim{
			Type:        ClaimType(tc.Name),
			Source:      "tool",
			Target:      tc.Target,
			Description: tc.Name + " " + tc.Target,
			Input:       rawToMap(tc.Input),
		}, true
	default:
		return Claim{}, false
	}
}

func rawToMap(in map[string]json.RawMessage) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		var val any
		if err := json.Unmarshal(v, &val); err == nil {
			out[k] = val
		}
	}
	return out
}

// IsActionClaim reports whether a claim type implies the agent took action.
func IsActionClaim(t ClaimType) bool {
	switch t {
	case ClaimTestPass, ClaimCommitted, ClaimPushed,
		ClaimFileCreated, ClaimFileModified, ClaimFileDeleted, ClaimCommandRan:
		return true
	default:
		return false
	}
}
