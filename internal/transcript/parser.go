package transcript

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

// ToolCall is a parsed Cursor tool_use block.
type ToolCall struct {
	ToolUseID string                     `json:"tool_use_id,omitempty"`
	Name      string                     `json:"name"`
	Input     map[string]json.RawMessage `json:"input"`
	Target    string                     `json:"target"`
	Result    string                     `json:"result,omitempty"`
	IsError   bool                       `json:"is_error,omitempty"`
}

// ToolResult is a parsed tool_result block before correlation.
type ToolResult struct {
	ToolUseID string
	Text      string
	IsError   bool
}

// ParsedLine is one parsed JSONL line.
type ParsedLine struct {
	Role        string
	Text        string
	ToolCalls   []ToolCall
	ToolResults []ToolResult
	TurnEnded   bool
	TurnStatus  string
}

type contentBlock struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	Text      string          `json:"text"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
}

type cursorMessage struct {
	Role    string `json:"role"`
	Message struct {
		Content []contentBlock `json:"content"`
	} `json:"message"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// ParseLines reads new JSONL content from path starting at fromOffset.
func ParseLines(path string, fromOffset int64) (lines []ParsedLine, newOffset int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fromOffset, err
	}
	defer f.Close()
	if _, err := f.Seek(fromOffset, 0); err != nil {
		return nil, fromOffset, err
	}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)
	off := fromOffset
	for scanner.Scan() {
		line := scanner.Text()
		off += int64(len(line)) + 1
		if strings.TrimSpace(line) == "" {
			continue
		}
		parsed, ok := parseLine(line)
		if !ok {
			continue
		}
		lines = append(lines, parsed)
	}
	return lines, off, scanner.Err()
}

func parseLine(line string) (ParsedLine, bool) {
	var row cursorMessage
	if err := json.Unmarshal([]byte(line), &row); err != nil {
		return ParsedLine{}, false
	}
	if row.Type == "turn_ended" {
		return ParsedLine{TurnEnded: true, TurnStatus: row.Status}, true
	}
	if row.Role == "" {
		return ParsedLine{}, false
	}
	var pl ParsedLine
	pl.Role = row.Role
	for _, c := range row.Message.Content {
		switch c.Type {
		case "text":
			if c.Text != "" {
				if pl.Text != "" {
					pl.Text += "\n"
				}
				pl.Text += c.Text
			}
		case "tool_use":
			tc := ToolCall{Name: c.Name, ToolUseID: c.ID}
			if len(c.Input) > 0 {
				_ = json.Unmarshal(c.Input, &tc.Input)
			}
			tc.Target = deriveTarget(tc)
			pl.ToolCalls = append(pl.ToolCalls, tc)
		case "tool_result":
			text := decodeToolResultContent(c.Content)
			pl.ToolResults = append(pl.ToolResults, ToolResult{
				ToolUseID: c.ToolUseID,
				Text:      text,
				IsError:   c.IsError,
			})
		}
	}
	if pl.Text == "" && len(pl.ToolCalls) == 0 && len(pl.ToolResults) == 0 {
		return ParsedLine{}, false
	}
	return pl, true
}

func decodeToolResultContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var b strings.Builder
		for i, block := range blocks {
			if block.Text == "" {
				continue
			}
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(block.Text)
			if i == len(blocks)-1 {
				break
			}
		}
		return b.String()
	}
	return string(raw)
}

func deriveTarget(tc ToolCall) string {
	if tc.Input == nil {
		return ""
	}
	for _, key := range []string{"path", "command", "glob_pattern"} {
		if raw, ok := tc.Input[key]; ok {
			var s string
			if err := json.Unmarshal(raw, &s); err == nil && s != "" {
				return s
			}
		}
	}
	return ""
}

// AttachToolResults correlates tool_result blocks to tool_use calls by ID.
func AttachToolResults(calls []ToolCall, results []ToolResult) []ToolCall {
	if len(results) == 0 {
		return calls
	}
	byID := make(map[string]ToolResult, len(results))
	for _, r := range results {
		if r.ToolUseID != "" {
			byID[r.ToolUseID] = r
		}
	}
	out := make([]ToolCall, len(calls))
	copy(out, calls)
	for i := range out {
		if out[i].ToolUseID == "" {
			continue
		}
		if r, ok := byID[out[i].ToolUseID]; ok {
			out[i].Result = r.Text
			out[i].IsError = r.IsError
		}
	}
	return out
}

// InputString returns a string field from tool input.
func InputString(tc ToolCall, key string) string {
	if tc.Input == nil {
		return ""
	}
	raw, ok := tc.Input[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}
