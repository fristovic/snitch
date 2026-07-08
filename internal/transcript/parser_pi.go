package transcript

import (
	"encoding/json"
)

// PiParser decodes Pi's (pi.dev) session transcript JSONL format.
//
// Location: ~/.pi/agent/sessions/--<encoded-path>--/<timestamp>_<uuid>.jsonl
//
// v3 layout: SessionMessageEntry lines wrap an AgentMessage in a nested
// "message" field ({type:"message", message:{role, content}}).
//
// Turn boundary: a user message begins a new turn (TurnEnded on user lines).
type PiParser struct{}

// Harness returns the pi harness identifier.
func (PiParser) Harness() string { return "pi" }

// piContent is one block of a Pi assistant message content array.
type piContent struct {
	Type       string          `json:"type"` // text | thinking | toolCall
	Text       string          `json:"text"`
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Arguments  json.RawMessage `json:"arguments"`
	ToolCallID string          `json:"toolCallId"`
}

// piAgentMessage is the nested AgentMessage inside type:"message" entries.
type piAgentMessage struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	ToolCallID string          `json:"toolCallId"`
	Output     string          `json:"output"`
	ExitCode   int             `json:"exitCode"`
	Command    string          `json:"command"`
	Model      string          `json:"model"` // model attribution on assistant messages
}

// piEntry is one line in a Pi session file.
type piEntry struct {
	Type    string          `json:"type"`
	Message *piAgentMessage `json:"message"`
}

// ParseLine decodes one Pi JSONL record into a normalized ParsedLine.
func (PiParser) ParseLine(line string) (ParsedLine, bool) {
	var e piEntry
	if err := json.Unmarshal([]byte(line), &e); err != nil {
		return ParsedLine{}, false
	}
	if e.Type != "message" || e.Message == nil {
		return ParsedLine{}, false
	}
	msg := e.Message
	blocks := decodePiBlocks(msg.Content)

	switch msg.Role {
	case "user":
		return ParsedLine{Role: "user", Text: piPlainText(blocks), TurnEnded: true}, true

	case "assistant":
		pl := ParsedLine{Role: "assistant", Model: msg.Model}
		for _, c := range blocks {
			switch c.Type {
			case "text":
				pl.Text = appendTextBlock(pl.Text, c.Text)
			case "thinking":
				// skip
			case "toolCall":
				tc := ToolCall{Name: canonicalToolName(c.Name, nil), ToolUseID: c.ID}
				if len(c.Arguments) > 0 {
					_ = json.Unmarshal(c.Arguments, &tc.Input)
				}
				tc.Target = deriveTarget(tc)
				pl.ToolCalls = append(pl.ToolCalls, tc)
			}
		}
		if pl.Text == "" && len(pl.ToolCalls) == 0 {
			return ParsedLine{}, false
		}
		return pl, true

	case "toolResult":
		text := msg.Output
		if text == "" {
			text = piPlainText(blocks)
		}
		if text == "" {
			return ParsedLine{}, false
		}
		return ParsedLine{
			Role: "assistant",
			ToolResults: []ToolResult{{
				ToolUseID: msg.ToolCallID,
				Text:      text,
			}},
		}, true

	case "bashExecution":
		tc := ToolCall{Name: ToolShell, Target: msg.Command, Result: msg.Output}
		if msg.ExitCode != 0 {
			tc.IsError = true
		}
		return ParsedLine{Role: "assistant", ToolCalls: []ToolCall{tc}}, true
	}

	return ParsedLine{}, false
}

func decodePiBlocks(raw json.RawMessage) []piContent {
	if len(raw) == 0 {
		return nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return []piContent{{Type: "text", Text: s}}
	}
	var blocks []piContent
	_ = json.Unmarshal(raw, &blocks)
	return blocks
}

func piPlainText(blocks []piContent) string {
	var text string
	for _, c := range blocks {
		if c.Type == "text" && c.Text != "" {
			text = appendTextBlock(text, c.Text)
		}
	}
	return text
}
