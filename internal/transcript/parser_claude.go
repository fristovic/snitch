package transcript

import (
	"encoding/json"
)

// ClaudeParser decodes Claude Code's transcript JSONL format.
//
// Location: ~/.claude/projects/<slug>/<session-uuid>.jsonl
// Each line is one message. Top-level type is "user" or "assistant". An
// assistant message carries message.content[] blocks (text / thinking /
// tool_use). Tool results arrive in subsequent "user" messages as
// tool_result blocks.
//
// Turn boundary: assistant message.stop_reason "end_turn" completes a turn.
// Real sessions store stop_reason on message (not top-level); both are checked.
//
// Tool name mapping (raw → canonical):
//
//	Bash → Shell, Edit → StrReplace, Agent → Task.
type ClaudeParser struct{}

// Harness returns the claude harness identifier.
func (ClaudeParser) Harness() string { return "claude" }

// claudeMessage is one JSONL record from ~/.claude/projects.
type claudeMessage struct {
	Type       string `json:"type"`        // "user" | "assistant"
	StopReason string `json:"stop_reason"` // legacy top-level (often null)
	Cwd        string `json:"cwd"`
	Message    struct {
		Role       string          `json:"role"`
		StopReason string          `json:"stop_reason"` // "end_turn" | "tool_use"
		Content    json.RawMessage `json:"content"`     // string or []contentBlock
	} `json:"message"`
}

// claudeToolNames maps Claude Code's PascalCase tool names to Snitch
// canonical names (layered over the shared lowercase map).
var claudeToolNames = map[string]string{
	"Bash":      ToolShell,
	"Write":     ToolWrite,
	"Edit":      ToolStrReplace,
	"MultiEdit": ToolStrReplace,
	"Read":      ToolRead,
	"Glob":      ToolGlob,
	"Grep":      ToolGrep,
	"Agent":     ToolTask,
	"Task":      ToolTask,
	// Web tools pass through under their own names deliberately — no verifier
	// handles them yet, and unknown tool names survive unchanged by design.
	"WebSearch": "WebSearch",
	"WebFetch":  "WebFetch",
}

// ParseLine decodes one Claude Code JSONL record into a normalized ParsedLine.
func (ClaudeParser) ParseLine(line string) (ParsedLine, bool) {
	var row claudeMessage
	if err := json.Unmarshal([]byte(line), &row); err != nil {
		return ParsedLine{}, false
	}

	pl := ParsedLine{Cwd: row.Cwd}

	switch row.Type {
	case "assistant":
		pl.Role = "assistant"
		blocks, plain := decodeClaudeContent(row.Message.Content)
		if plain != "" {
			pl.Text = plain
		}
		for _, c := range blocks {
			switch c.Type {
			case "text":
				pl.Text = appendTextBlock(pl.Text, c.Text)
			case "thinking":
				// Skip — thinking blocks are not prose claims.
			case "tool_use":
				tc := NewToolCall(c.Name, claudeToolNames)
				tc.ToolUseID = c.ID
				if len(c.Input) > 0 {
					_ = json.Unmarshal(c.Input, &tc.Input)
				}
				tc.Target = deriveTarget(tc)
				pl.ToolCalls = append(pl.ToolCalls, tc)
			}
		}
		stop := row.Message.StopReason
		if stop == "" {
			stop = row.StopReason
		}
		if stop == "end_turn" {
			pl.TurnEnded = true
		}
		if pl.Text == "" && len(pl.ToolCalls) == 0 && !pl.TurnEnded {
			return ParsedLine{}, false
		}
		return pl, true

	case "user":
		// User lines carry real user prose AND tool_result blocks (Claude
		// delivers results on user-role messages). Role stays "user"; the
		// turn assembler attaches ToolResults regardless of role.
		pl.Role = "user"
		blocks, plain := decodeClaudeContent(row.Message.Content)
		if plain != "" {
			pl.Text = plain
		}
		for _, c := range blocks {
			switch c.Type {
			case "tool_result":
				pl.ToolResults = append(pl.ToolResults, ToolResult{
					ToolUseID: c.ToolUseID,
					Text:      decodeToolResultContent(c.Content),
					IsError:   c.IsError,
				})
			case "text":
				pl.Text = appendTextBlock(pl.Text, c.Text)
			}
		}
		if pl.Text == "" && len(pl.ToolResults) == 0 {
			return ParsedLine{}, false
		}
		return pl, true
	}

	return ParsedLine{}, false
}

// decodeClaudeContent parses message.content as either a plain string or a
// content-block array (Claude Code uses both shapes on real sessions).
func decodeClaudeContent(raw json.RawMessage) (blocks []contentBlock, plain string) {
	if len(raw) == 0 {
		return nil, ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return nil, s
	}
	_ = json.Unmarshal(raw, &blocks)
	return blocks, ""
}
