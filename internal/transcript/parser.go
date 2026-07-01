package transcript

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

// ToolCall is a parsed Cursor tool_use block.
type ToolCall struct {
	Name   string                     `json:"name"`
	Input  map[string]json.RawMessage `json:"input"`
	Target string                     `json:"target"`
}

// ParsedLine is one parsed JSONL line.
type ParsedLine struct {
	Role       string
	Text       string
	ToolCalls  []ToolCall
	TurnEnded  bool
	TurnStatus string
}

type contentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
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
			tc := ToolCall{Name: c.Name}
			if len(c.Input) > 0 {
				_ = json.Unmarshal(c.Input, &tc.Input)
			}
			tc.Target = deriveTarget(tc)
			pl.ToolCalls = append(pl.ToolCalls, tc)
		}
	}
	if pl.Text == "" && len(pl.ToolCalls) == 0 {
		return ParsedLine{}, false
	}
	return pl, true
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
