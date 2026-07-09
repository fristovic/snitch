// Package claims holds claim-type labels and display helpers.
// Canonical claim_type ID strings are owned by internal/verify/verifiers;
// this package mirrors them for UI labels and normalization.
package claims

import "strings"

// Canonical claim_type values (snake_case). Keep in sync with verifiers.Claim*.
const (
	TypeTestPass          = "test_pass"
	TypeCommitted         = "committed"
	TypePushed            = "pushed"
	TypeFileCreated       = "file_created"
	TypeFileModified      = "file_modified"
	TypeFileDeleted       = "file_deleted"
	TypeCommandRan        = "command_ran"
	TypeCommandSucceeded  = "command_succeeded"
	TypeStub              = "stub"
	TypeNoAction          = "no_action"
	TypeSelfContradiction = "self_contradiction"
	TypeCountMismatch     = "count_mismatch"
	TypeNegationViolation = "negation_violation"

	TypeToolWrite      = "tool_write"
	TypeToolStrReplace = "tool_str_replace"
	TypeToolDelete     = "tool_delete"
	TypeToolRead       = "tool_read"
	TypeToolGlob       = "tool_glob"
	TypeToolShell      = "tool_shell"
	TypeToolTask       = "tool_task"

	TypeMissed = "missed"
)

// toolMeta is the single registry for harness tool names → claim_type + label.
type toolMeta struct {
	Name  string // harness tool name (Write, Shell, …)
	Type  string // tool_* claim_type
	Label string
}

var tools = []toolMeta{
	{Name: "Write", Type: TypeToolWrite, Label: "File write"},
	{Name: "StrReplace", Type: TypeToolStrReplace, Label: "File edit"},
	{Name: "Delete", Type: TypeToolDelete, Label: "File delete"},
	{Name: "Read", Type: TypeToolRead, Label: "File read"},
	{Name: "Glob", Type: TypeToolGlob, Label: "File search"},
	{Name: "Shell", Type: TypeToolShell, Label: "Shell command"},
	{Name: "Task", Type: TypeToolTask, Label: "Subagent task"},
}

var (
	toolNameToType   map[string]string
	legacyToolTypes  map[string]string // PascalCase claim_type → tool_*
	toolClaimLabels  map[string]string
	toolClaimPrefixes []string
)

func init() {
	toolNameToType = make(map[string]string, len(tools))
	legacyToolTypes = make(map[string]string, len(tools))
	toolClaimLabels = make(map[string]string, len(tools)*2)
	for _, t := range tools {
		toolNameToType[t.Name] = t.Type
		legacyToolTypes[t.Name] = t.Type
		toolClaimLabels[t.Type] = t.Label
		toolClaimLabels[t.Name] = t.Label // legacy PascalCase rows
		toolClaimPrefixes = append(toolClaimPrefixes, t.Name+" ", t.Type+" ")
	}
}

// ToolNameToClaimType maps a canonical tool name (Write, Shell, …) to tool_*.
// Prefer verifiers.ToolNameToClaimType at verification boundaries.
func ToolNameToClaimType(toolName string) (string, bool) {
	t, ok := toolNameToType[toolName]
	return t, ok
}

// NormalizeClaimType returns a snake_case claim_type. Legacy PascalCase tool
// types are rewritten to tool_*; already-normalized IDs pass through.
func NormalizeClaimType(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if n, ok := legacyToolTypes[raw]; ok {
		return n
	}
	return raw
}

// AllFilterTypes returns claim types for dashboard filter cycling, ordered
// prose → consistency → tool. Callers typically prepend "" for "all".
func AllFilterTypes() []string {
	return []string{
		TypeTestPass,
		TypeCommitted,
		TypePushed,
		TypeFileCreated,
		TypeFileModified,
		TypeFileDeleted,
		TypeCommandRan,
		TypeCommandSucceeded,
		TypeStub,
		TypeNoAction,
		TypeSelfContradiction,
		TypeCountMismatch,
		TypeNegationViolation,
		TypeToolWrite,
		TypeToolStrReplace,
		TypeToolDelete,
		TypeToolRead,
		TypeToolGlob,
		TypeToolShell,
		TypeToolTask,
	}
}
