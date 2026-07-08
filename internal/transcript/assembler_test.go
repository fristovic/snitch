package transcript

import (
	"testing"
	"time"
)

// stubResolver returns a fixed project cwd for assembler provenance tests.
type stubResolver struct{ cwd string }

func (s stubResolver) ProjectCwd(string) string { return s.cwd }
func (s stubResolver) ProjectDir(string) string { return "" }
func (s stubResolver) SessionID(string) string  { return "sess" }

func newTestAssembler(t *testing.T) *turnAssembler {
	t.Helper()
	return newTurnAssembler(stubResolver{cwd: t.TempDir()}, "/tmp/x.jsonl")
}

// Cursor/Codex: explicit marker line with no role flushes the buffer.
func TestAssemblerMarkerBoundary(t *testing.T) {
	a := newTestAssembler(t)
	if buf := a.Feed(ParsedLine{Role: "user", Text: "do it"}); buf != nil {
		t.Fatal("user line should not complete a turn")
	}
	if buf := a.Feed(ParsedLine{Role: "assistant", Text: "done", ToolCalls: []ToolCall{{Name: ToolShell}}}); buf != nil {
		t.Fatal("assistant line should not complete a turn")
	}
	buf := a.Feed(ParsedLine{TurnEnded: true})
	if buf == nil {
		t.Fatal("marker should complete the turn")
	}
	if buf.userText != "do it" || buf.assistantText.String() != "done" || len(buf.toolCalls) != 1 {
		t.Fatalf("wrong turn content: %+v", buf)
	}
	if a.buf != nil {
		t.Fatal("assembler should be reset after flush")
	}
}

// Claude: the final assistant line carries content AND TurnEnded.
func TestAssemblerClaudeEndTurn(t *testing.T) {
	a := newTestAssembler(t)
	a.Feed(ParsedLine{Role: "user", Text: "fix the bug"})
	// Tool results ride on user-role lines and attach regardless of role.
	a.Feed(ParsedLine{Role: "assistant", ToolCalls: []ToolCall{{Name: ToolShell, ToolUseID: "t1"}}})
	a.Feed(ParsedLine{Role: "user", ToolResults: []ToolResult{{ToolUseID: "t1", Text: "ok"}}})
	buf := a.Feed(ParsedLine{Role: "assistant", Text: "Fixed.", TurnEnded: true})
	if buf == nil {
		t.Fatal("end_turn assistant line should complete the turn")
	}
	if buf.assistantText.String() != "Fixed." {
		t.Fatalf("assistant text %q", buf.assistantText.String())
	}
	attached := AttachToolResults(buf.toolCalls, buf.toolResults)
	if len(attached) != 1 || attached[0].Result != "ok" {
		t.Fatalf("tool result not attached: %+v", attached)
	}
}

// Claude: user prose on a tool_result carrier line must land in userText.
func TestAssemblerUserProseWithToolResult(t *testing.T) {
	a := newTestAssembler(t)
	a.Feed(ParsedLine{Role: "user", Text: "please continue", ToolResults: []ToolResult{{ToolUseID: "t1", Text: "output"}}})
	buf := a.Feed(ParsedLine{Role: "assistant", Text: "on it", TurnEnded: true})
	if buf == nil {
		t.Fatal("expected completed turn")
	}
	if buf.userText != "please continue" {
		t.Fatalf("user prose lost: %q", buf.userText)
	}
	if len(buf.toolResults) != 1 {
		t.Fatalf("tool result lost: %+v", buf.toolResults)
	}
}

// Pi: a user line with TurnEnded flushes the prior turn and carries its text
// into a fresh buffer with full provenance.
func TestAssemblerPiUserCarry(t *testing.T) {
	a := newTestAssembler(t)
	if buf := a.Feed(ParsedLine{Role: "user", Text: "first", TurnEnded: true}); buf != nil {
		t.Fatal("empty prior buffer should not emit")
	}
	a.Feed(ParsedLine{Role: "assistant", Text: "answer one"})
	buf := a.Feed(ParsedLine{Role: "user", Text: "second", TurnEnded: true})
	if buf == nil {
		t.Fatal("second user message should flush the first turn")
	}
	if buf.userText != "first" || buf.assistantText.String() != "answer one" {
		t.Fatalf("wrong first turn: user=%q assistant=%q", buf.userText, buf.assistantText.String())
	}
	// The carried buffer must have provenance, not a bare struct.
	if a.buf == nil || a.buf.userText != "second" {
		t.Fatalf("carry buffer wrong: %+v", a.buf)
	}
	if a.buf.projectPath == "" {
		t.Fatal("carry buffer lost projectPath provenance")
	}
	if a.buf.startedAt.IsZero() {
		t.Fatal("carry buffer lost startedAt")
	}
}

func TestAssemblerIdleFlush(t *testing.T) {
	a := newTestAssembler(t)
	a.Feed(ParsedLine{Role: "assistant", Text: "trailing turn with no end marker"})
	if buf := a.Idle(time.Now().Add(-time.Minute)); buf != nil {
		t.Fatal("recently-written buffer must not idle-flush")
	}
	a.buf.lastWriteAt = time.Now().Add(-time.Hour)
	buf := a.Idle(time.Now().Add(-time.Minute))
	if buf == nil {
		t.Fatal("stale buffer should idle-flush")
	}
	if buf.assistantText.String() != "trailing turn with no end marker" {
		t.Fatalf("wrong idle content: %q", buf.assistantText.String())
	}
	if a.buf != nil {
		t.Fatal("assembler should be cleared after idle flush")
	}
}

func TestAssemblerDrain(t *testing.T) {
	a := newTestAssembler(t)
	if buf := a.Drain(); buf != nil {
		t.Fatal("empty assembler should drain nil")
	}
	a.Feed(ParsedLine{Role: "user", Text: "hello"})
	buf := a.Drain()
	if buf == nil || buf.userText != "hello" {
		t.Fatalf("drain lost content: %+v", buf)
	}
	if a.buf != nil {
		t.Fatal("assembler should be cleared after drain")
	}
}

// Cwd hints (Codex session_meta, Claude per-event cwd) apply to the buffer.
func TestAssemblerCwdHint(t *testing.T) {
	a := newTurnAssembler(stubResolver{cwd: ""}, "/tmp/x.jsonl")
	a.Feed(ParsedLine{Cwd: "/proj/from/meta"})
	a.Feed(ParsedLine{Role: "assistant", Text: "hi"})
	buf := a.Feed(ParsedLine{TurnEnded: true})
	if buf == nil || buf.projectPath != "/proj/from/meta" {
		t.Fatalf("cwd hint not applied: %+v", buf)
	}
}
