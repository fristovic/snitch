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
	ClaimTestPass          ClaimType = "test_pass"
	ClaimCommitted         ClaimType = "committed"
	ClaimPushed            ClaimType = "pushed"
	ClaimFileCreated       ClaimType = "file_created"
	ClaimFileModified      ClaimType = "file_modified"
	ClaimFileDeleted       ClaimType = "file_deleted"
	ClaimCommandRan        ClaimType = "command_ran"
	ClaimCommandSucceeded  ClaimType = "command_succeeded"
	ClaimStub              ClaimType = "stub"
	ClaimNoAction          ClaimType = "no_action"
	ClaimSelfContradiction ClaimType = "self_contradiction"
	ClaimCountMismatch     ClaimType = "count_mismatch"
	ClaimNegationViolation ClaimType = "negation_violation"

	ClaimToolWrite      ClaimType = "tool_write"
	ClaimToolStrReplace ClaimType = "tool_str_replace"
	ClaimToolDelete     ClaimType = "tool_delete"
	ClaimToolRead       ClaimType = "tool_read"
	ClaimToolGlob       ClaimType = "tool_glob"
	ClaimToolShell      ClaimType = "tool_shell"
	ClaimToolTask       ClaimType = "tool_task"
)

// toolNameToType maps canonical harness tool names to claim_type IDs.
var toolNameToType = map[string]ClaimType{
	"Write":      ClaimToolWrite,
	"StrReplace": ClaimToolStrReplace,
	"Delete":     ClaimToolDelete,
	"Read":       ClaimToolRead,
	"Glob":       ClaimToolGlob,
	"Shell":      ClaimToolShell,
	"Task":       ClaimToolTask,
}

// ToolNameToClaimType maps a canonical tool name to tool_*.
func ToolNameToClaimType(toolName string) (ClaimType, bool) {
	t, ok := toolNameToType[toolName]
	return t, ok
}

// IsLocalActionType reports claim types that imply the agent mutated local state.
func IsLocalActionType(t ClaimType) bool {
	switch t {
	case ClaimCommitted, ClaimPushed,
		ClaimFileCreated, ClaimFileModified, ClaimFileDeleted,
		ClaimCommandRan, ClaimCommandSucceeded, ClaimStub:
		return true
	default:
		return false
	}
}

// Claim is a prose or tool-derived claim to verify.
type Claim struct {
	Type        ClaimType      `json:"type"`
	Source      string         `json:"source"`
	Target      string         `json:"target"`
	Quote       string         `json:"quote"`
	Description string         `json:"description"`
	Segment     string         `json:"segment,omitempty"` // execution | recap
	Confidence  int            `json:"confidence,omitempty"`
	Input       map[string]any `json:"input,omitempty"`
	// Sentence is the full assistant sentence containing Quote (prose only).
	Sentence string `json:"sentence,omitempty"`
	// Context is a capped ±1–2 sentence window around Sentence (prose only).
	Context string `json:"context,omitempty"`
}

// TurnEvidence is a prior turn's checkable evidence.
type TurnEvidence struct {
	RunID        string
	ToolCalls    []transcript.ToolCall
	StartHEAD    string
	EndHEAD      string
	StartedAt    time.Time
	FinishedAt   time.Time
	FileManifest map[string]string
}

// VerifyContext provides runtime context for verification.
type VerifyContext struct {
	Output             string
	Cwd                string
	ProjectPath        string
	Harness            string
	StartHEAD          string
	EndHEAD            string
	FileManifest       map[string]string
	TranscriptPath     string
	ObservedAt         time.Time
	StartedAt          time.Time
	FinishedAt         time.Time
	ToolCalls          []transcript.ToolCall
	EffectiveToolCalls []transcript.ToolCall
	PriorTurns         []TurnEvidence
	ExecutionText      string
	RecapText          string
	AssistantText      string
	// ShellOutputResolver resolves shell output from harness-specific
	// artifacts (Cursor terminal files). nil means inline tool results only.
	ShellOutputResolver transcript.ShellOutputResolver
}

// AllToolCalls returns merged current-turn tool calls including subagent evidence.
func AllToolCalls(ctx VerifyContext) []transcript.ToolCall {
	if len(ctx.EffectiveToolCalls) > 0 {
		return ctx.EffectiveToolCalls
	}
	return ctx.ToolCalls
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

// ToolCallToClaim maps a tool call (already normalized to canonical names by
// the harness parser) to a verifiable claim. claim_type uses tool_* snake_case.
func ToolCallToClaim(tc transcript.ToolCall) (Claim, bool) {
	claimType, ok := ToolNameToClaimType(tc.Name)
	if !ok {
		return Claim{}, false
	}
	return Claim{
		Type:        claimType,
		Source:      "tool",
		Target:      tc.Target,
		Description: tc.Name + " " + tc.Target,
		Input:       rawToMap(tc.Input),
	}, true
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
