package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

const subagentWindowSlack = 30 * time.Second

// LoadSubagentToolCalls returns tool calls from subagent transcripts finished within the parent turn window.
func LoadSubagentToolCalls(parentPath string, windowStart, windowEnd time.Time) ([]ToolCall, error) {
	if parentPath == "" {
		return nil, nil
	}
	subDir := filepath.Join(filepath.Dir(parentPath), "subagents")
	entries, err := os.ReadDir(subDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	windowEnd = windowEnd.Add(subagentWindowSlack)

	var merged []ToolCall
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil || info.Size() == 0 {
			continue
		}
		path := filepath.Join(subDir, e.Name())
		calls, err := toolCallsInWindow(path, windowStart, windowEnd)
		if err != nil {
			continue
		}
		for i := range calls {
			if calls[i].ToolUseID != "" {
				calls[i].ToolUseID = "subagent:" + calls[i].ToolUseID
			} else {
				calls[i].ToolUseID = "subagent:" + e.Name() + ":" + calls[i].Name
			}
			merged = append(merged, calls[i])
		}
	}
	return merged, nil
}

func toolCallsInWindow(path string, windowStart, windowEnd time.Time) ([]ToolCall, error) {
	lines, _, err := ParseLines(path, 0)
	if err != nil {
		return nil, err
	}
	var buf struct {
		toolCalls   []ToolCall
		toolResults []ToolResult
		startedAt   time.Time
	}
	var out []ToolCall
	for _, line := range lines {
		if line.TurnEnded {
			finishedAt := time.Now()
			if !buf.startedAt.IsZero() && inSubagentWindow(buf.startedAt, finishedAt, windowStart, windowEnd) {
				out = append(out, AttachToolResults(buf.toolCalls, buf.toolResults)...)
			}
			buf.toolCalls = nil
			buf.toolResults = nil
			buf.startedAt = time.Time{}
			continue
		}
		if buf.startedAt.IsZero() {
			buf.startedAt = time.Now()
		}
		buf.toolCalls = append(buf.toolCalls, line.ToolCalls...)
		buf.toolResults = append(buf.toolResults, line.ToolResults...)
	}
	return out, nil
}

func inSubagentWindow(turnStart, turnEnd, windowStart, windowEnd time.Time) bool {
	if windowStart.IsZero() || windowEnd.IsZero() {
		return true
	}
	return !turnEnd.Before(windowStart) && !turnStart.After(windowEnd)
}
