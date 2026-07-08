package transcript_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/fristovic/snitch/internal/transcript"
)

// TestClaudeParser covers text, tool_use, tool_result, thinking exclusion,
// stop_reason turn-end, and Bash→Shell / Edit→StrReplace normalization.
func TestClaudeParser(t *testing.T) {
	path := filepath.Join("..", "..", "test", "fixtures", "sample_transcripts", "claude_assistant.jsonl")
	lines, _, err := transcript.ParseLinesWith(transcript.ClaudeParser{}, path, 0)
	if err != nil {
		t.Fatal(err)
	}

	var (
		assistantText  string
		toolCalls      []transcript.ToolCall
		toolResults    []transcript.ToolResult
		turnEnds       int
		thinkingLeaked bool
	)
	for _, l := range lines {
		if l.TurnEnded {
			turnEnds++
		}
		if l.Role == "assistant" {
			assistantText += l.Text
			toolCalls = append(toolCalls, l.ToolCalls...)
		}
		// Tool results ride on user-role lines in Claude's format; the turn
		// assembler attaches them regardless of the carrier line's role.
		toolResults = append(toolResults, l.ToolResults...)
	}

	// Thinking block text ("planning my approach") must never appear.
	if strings.Contains(assistantText, "planning my approach") {
		thinkingLeaked = true
	}
	if thinkingLeaked {
		t.Fatal("thinking block text leaked into assistant prose")
	}

	// One end_turn marker in the fixture.
	if turnEnds != 1 {
		t.Fatalf("expected 1 turn end, got %d", turnEnds)
	}

	// Bash→Shell and Edit→StrReplace normalization, plus inline results.
	if !hasToolNamed(toolCalls, "Shell") {
		t.Errorf("expected Bash normalized to Shell; calls=%+v", toolCalls)
	}
	if !hasToolNamed(toolCalls, "StrReplace") {
		t.Errorf("expected Edit normalized to StrReplace; calls=%+v", toolCalls)
	}

	// The bash tool_result (is_error=true, "3 failing") should correlate.
	var foundErrResult bool
	for _, r := range toolResults {
		if r.IsError {
			foundErrResult = true
		}
	}
	if !foundErrResult {
		t.Errorf("expected an error tool_result from the Bash call; results=%+v", toolResults)
	}
}

// TestCodexParser covers session_meta skip, turn_context markers, function_call
// (shell→Shell, apply_patch→StrReplace), and function_call_output correlation.
func TestCodexParser(t *testing.T) {
	path := filepath.Join("..", "..", "test", "fixtures", "sample_transcripts", "codex_rollout.jsonl")
	lines, _, err := transcript.ParseLinesWith(transcript.CodexParser{}, path, 0)
	if err != nil {
		t.Fatal(err)
	}

	var (
		toolCalls   []transcript.ToolCall
		toolResults []transcript.ToolResult
		turnEnds    int
		sawUserText bool
	)
	for _, l := range lines {
		if l.TurnEnded {
			turnEnds++
		}
		if l.Role == "user" && l.Text != "" {
			sawUserText = true
		}
		toolCalls = append(toolCalls, l.ToolCalls...)
		toolResults = append(toolResults, l.ToolResults...)
	}

	if !sawUserText {
		t.Error("expected user message text from input_text content")
	}
	// Two turn_context markers in the fixture.
	if turnEnds != 2 {
		t.Fatalf("expected 2 turn boundaries, got %d", turnEnds)
	}
	if !hasToolNamed(toolCalls, "Shell") {
		t.Errorf("expected shell→Shell; calls=%+v", toolCalls)
	}
	if !hasToolNamed(toolCalls, "StrReplace") {
		t.Errorf("expected apply_patch→StrReplace; calls=%+v", toolCalls)
	}
	// function_call_output should surface as a tool result for call_01.
	if !hasResultFor(toolResults, "call_01") {
		t.Errorf("expected function_call_output correlated to call_01; results=%+v", toolResults)
	}
}

// TestPiParser covers session-header skip, toolCall (bash→Shell), toolResult
// correlation, bashExecution messages, and user-message turn ends.
func TestPiParser(t *testing.T) {
	path := filepath.Join("..", "..", "test", "fixtures", "sample_transcripts", "pi_session.jsonl")
	lines, _, err := transcript.ParseLinesWith(transcript.PiParser{}, path, 0)
	if err != nil {
		t.Fatal(err)
	}

	var (
		toolCalls []transcript.ToolCall
		turnEnds  int
	)
	for _, l := range lines {
		if l.TurnEnded {
			turnEnds++
		}
		toolCalls = append(toolCalls, l.ToolCalls...)
	}

	// bash toolCall normalized to Shell; bashExecution also yields a Shell call.
	if !hasToolNamed(toolCalls, "Shell") {
		t.Errorf("expected bash→Shell; calls=%+v", toolCalls)
	}
	// The user message sets TurnEnded for the prior turn; here it's the first
	// line so it flushes an empty buffer (still counts as a TurnEnded line).
	if turnEnds < 1 {
		t.Errorf("expected at least 1 user-message turn end, got %d", turnEnds)
	}
}

// TestClaudePathResolver covers cwd derivation from Claude and Cursor slugs.
func TestClaudePathResolver(t *testing.T) {
	r := transcript.ClaudePathResolver{}
	path := "/Users/alice/.claude/projects/Users-alice-code-app/abc-123.jsonl"
	if cwd := r.ProjectCwd(path); cwd != "/Users/alice/code/app" {
		t.Fatalf("ProjectCwd got %q", cwd)
	}
	claudePath := "/Users/alice/.claude/projects/-Users-alice-Desktop-app/abc-123.jsonl"
	if cwd := r.ProjectCwd(claudePath); cwd != "/Users/alice/Desktop/app" {
		t.Fatalf("Claude slug ProjectCwd got %q", cwd)
	}
	if sid := r.SessionID(path); sid != "abc-123" {
		t.Fatalf("SessionID got %q", sid)
	}
}

// TestCodexPathResolver covers session-id extraction from rollout filenames.
func TestCodexPathResolver(t *testing.T) {
	r := transcript.CodexPathResolver{}
	path := "/home/u/.codex/sessions/2026/07/08/rollout-12345-deadbeef.jsonl"
	if sid := r.SessionID(path); sid != "12345-deadbeef" {
		t.Fatalf("SessionID got %q", sid)
	}
}

// TestPiPathResolver covers cwd derivation from the --...-- path encoding.
func TestPiPathResolver(t *testing.T) {
	r := transcript.PiPathResolver{}
	path := "/home/u/.pi/agent/sessions/--Users-alice-proj--/ts_uuid.jsonl"
	if cwd := r.ProjectCwd(path); cwd != "/Users/alice/proj" {
		t.Fatalf("ProjectCwd got %q", cwd)
	}
}

func hasToolNamed(calls []transcript.ToolCall, name string) bool {
	for _, c := range calls {
		if c.Name == name {
			return true
		}
	}
	return false
}

func hasResultFor(results []transcript.ToolResult, id string) bool {
	for _, r := range results {
		if r.ToolUseID == id {
			return true
		}
	}
	return false
}
