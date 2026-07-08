package transcript

import (
	"encoding/json"
)

// CodexParser decodes OpenAI Codex CLI's session rollout JSONL format.
//
// Location: ~/.codex/sessions/YYYY/MM/DD/rollout-<ts>-<uuid>.jsonl
// (CODEX_HOME overrides ~/.codex).
//
// Each line is a RolloutLine: {timestamp, type, payload}. The first line is
// session_meta (cwd). response_item payloads carry messages, function_call,
// and function_call_output. turn_context marks explicit turn boundaries.
//
// Tool name mapping (raw → canonical):
//
//	shell → Shell, apply_patch → StrReplace, read_file → Read, etc.
type CodexParser struct{}

// Harness returns the codex harness identifier.
func (CodexParser) Harness() string { return "codex" }

// codexLine is the top-level RolloutLine wrapper.
type codexLine struct {
	Type    string          `json:"type"` // session_meta | response_item | turn_context
	Payload json.RawMessage `json:"payload"`
}

// codexContentItem is one element of a Message content array.
type codexContentItem struct {
	Type string `json:"type"` // "input_text" | "output_text" | "text"
	Text string `json:"text"`
}

// codexToolNames maps Codex-specific tool names to Snitch canonical names
// (layered over the shared lowercase map, which covers shell/grep/task).
var codexToolNames = map[string]string{
	"run_shell":   ToolShell,
	"apply_patch": ToolStrReplace,
	"read_file":   ToolRead,
	"write_file":  ToolWrite,
	"ls":          ToolGlob,
}

// ParseLine decodes one Codex rollout JSONL record into a normalized ParsedLine.
func (CodexParser) ParseLine(line string) (ParsedLine, bool) {
	var row codexLine
	if err := json.Unmarshal([]byte(line), &row); err != nil {
		return ParsedLine{}, false
	}

	switch row.Type {
	case "session_meta":
		var meta struct {
			Cwd string `json:"cwd"`
		}
		_ = json.Unmarshal(row.Payload, &meta)
		if meta.Cwd == "" {
			return ParsedLine{}, false
		}
		return ParsedLine{Cwd: meta.Cwd}, true
	case "turn_context":
		return ParsedLine{TurnEnded: true}, true
	case "response_item":
		return parseCodexResponseItem(row.Payload)
	}
	return ParsedLine{}, false
}

// parseCodexResponseItem decodes a Codex ResponseItem JSON blob.
func parseCodexResponseItem(raw []byte) (ParsedLine, bool) {
	if len(raw) == 0 {
		return ParsedLine{}, false
	}
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return ParsedLine{}, false
	}

	switch head.Type {
	case "message":
		var msg struct {
			Role    string             `json:"role"`
			Content []codexContentItem `json:"content"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			return ParsedLine{}, false
		}
		var text string
		for _, c := range msg.Content {
			text = appendTextBlock(text, c.Text)
		}
		if text == "" {
			return ParsedLine{}, false
		}
		role := msg.Role
		if role == "" {
			role = "assistant"
		}
		return ParsedLine{Role: role, Text: text}, true

	case "function_call":
		var fc struct {
			Name      string          `json:"name"`
			CallID    string          `json:"call_id"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(raw, &fc); err != nil {
			return ParsedLine{}, false
		}
		tc := ToolCall{Name: canonicalToolName(fc.Name, codexToolNames), ToolUseID: fc.CallID}
		if len(fc.Arguments) > 0 {
			decodeArgsInto(fc.Arguments, &tc.Input)
		}
		tc.Target = deriveTarget(tc)
		return ParsedLine{Role: "assistant", ToolCalls: []ToolCall{tc}}, true

	case "function_call_output":
		var out struct {
			CallID string `json:"call_id"`
			Output string `json:"output"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return ParsedLine{}, false
		}
		return ParsedLine{
			Role: "assistant",
			ToolResults: []ToolResult{{
				ToolUseID: out.CallID,
				Text:      out.Output,
			}},
		}, true
	}

	return ParsedLine{}, false
}

// decodeArgsInto decodes a tool-call arguments blob into an input map.
func decodeArgsInto(raw []byte, out *map[string]json.RawMessage) {
	if len(raw) == 0 {
		return
	}
	if err := json.Unmarshal(raw, out); err == nil {
		return
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		_ = json.Unmarshal([]byte(s), out)
	}
}
