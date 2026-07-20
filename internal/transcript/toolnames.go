package transcript

// Canonical tool name constants. These are Snitch's internal tool vocabulary —
// the names the verify layer matches on across its verifiers. Each harness
// parser normalizes its raw tool names to these at parse time via a
// per-harness map. Verifiers therefore never need to know which platform
// produced a tool call.
const (
	ToolShell      = "Shell"
	ToolWrite      = "Write"
	ToolStrReplace = "StrReplace"
	ToolDelete     = "Delete"
	ToolRead       = "Read"
	ToolGlob       = "Glob"
	ToolGrep       = "Grep"
	ToolTask       = "Task"
)

// lowercaseToolNames is the shared raw→canonical map for harnesses that use
// lowercase tool names (Pi, OpenCode, and most of Codex). Harnesses with
// additional or differently-cased names (Claude PascalCase, Codex
// apply_patch/read_file/...) layer their own entries on top.
var lowercaseToolNames = map[string]string{
	"bash":   ToolShell,
	"shell":  ToolShell,
	"write":  ToolWrite,
	"read":   ToolRead,
	"edit":   ToolStrReplace,
	"delete": ToolDelete,
	"grep":   ToolGrep,
	"glob":   ToolGlob,
	"task":   ToolTask,
}

// canonicalToolName resolves a raw harness tool name through overrides first,
// then the shared lowercase map, passing unknown names through unchanged.
func canonicalToolName(raw string, overrides map[string]string) string {
	if name, ok := overrides[raw]; ok {
		return name
	}
	if name, ok := lowercaseToolNames[raw]; ok {
		return name
	}
	return raw
}

// NewToolCall builds a ToolCall with RawName set to the harness-native name and
// Name set to the Snitch canonical vocabulary.
func NewToolCall(raw string, overrides map[string]string) ToolCall {
	return ToolCall{
		RawName: raw,
		Name:    canonicalToolName(raw, overrides),
	}
}
