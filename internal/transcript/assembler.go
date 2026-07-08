package transcript

import (
	"strings"
	"time"
)

// turnBuffer accumulates one in-progress turn's content.
type turnBuffer struct {
	userText      string
	assistantText strings.Builder
	toolCalls     []ToolCall
	toolResults   []ToolResult
	startedAt     time.Time
	startHEAD     string
	projectPath   string
	model         string
	lastWriteAt   time.Time
}

func (b *turnBuffer) empty() bool {
	return b.userText == "" && b.assistantText.Len() == 0 && len(b.toolCalls) == 0
}

// turnAssembler folds ParsedLines into turn buffers and decides turn
// boundaries. It is the single place that encodes every harness's boundary
// semantics:
//
//   - Cursor/Codex/OpenCode: an explicit marker line (turn_ended /
//     turn_context / finish=="stop" end marker) with no role — flush after
//     appending nothing.
//   - Claude: the final assistant line carries both content and TurnEnded —
//     append content first, then flush.
//   - Pi/OpenCode: a user line with TurnEnded STARTS the next turn — flush
//     the prior buffer and seed a fresh one carrying the user text.
//
// The assembler is synchronous and unlocked; callers serialize access (the
// Watcher via its event loop, the OpenCode reader by building a fresh
// assembler per poll).
type turnAssembler struct {
	resolver PathResolver
	path     string
	buf      *turnBuffer
}

func newTurnAssembler(resolver PathResolver, path string) *turnAssembler {
	return &turnAssembler{resolver: resolver, path: path}
}

// newBuffer creates a buffer with full provenance (project path + start HEAD),
// used for both initial buffers and Pi user-carry successors.
func (a *turnAssembler) newBuffer() *turnBuffer {
	projectPath := a.resolver.ProjectCwd(a.path)
	return &turnBuffer{
		startedAt:   time.Now(),
		startHEAD:   GitHEAD(projectPath),
		projectPath: projectPath,
		lastWriteAt: time.Now(),
	}
}

// Feed processes one parsed line. It returns a completed turn buffer when the
// line closes a turn, or nil when the turn is still accumulating.
func (a *turnAssembler) Feed(line ParsedLine) *turnBuffer {
	// Event time: sources with recorded timestamps (OpenCode DB rows) use
	// them so turn times and poll cursors reflect reality; live watchers
	// stamp wall-clock time.
	now := line.Timestamp
	if now.IsZero() {
		now = time.Now()
	}
	created := a.buf == nil
	if created {
		a.buf = a.newBuffer()
	}
	buf := a.buf
	buf.lastWriteAt = now
	if created {
		buf.startedAt = now
	}

	if line.Cwd != "" {
		if buf.projectPath == "" {
			buf.projectPath = line.Cwd
		}
		if buf.startHEAD == "" {
			buf.startHEAD = GitHEAD(buf.projectPath)
		}
	}
	if line.Model != "" {
		buf.model = line.Model
	}

	// Pi/OpenCode: a user line that signals TurnEnded is the START of the
	// next turn — flush the prior buffer and carry the user text into a
	// fresh one.
	if line.TurnEnded && line.Role == "user" {
		next := a.newBuffer()
		next.startedAt = now
		next.lastWriteAt = now
		next.userText = line.Text
		next.toolResults = append(next.toolResults, line.ToolResults...)
		a.buf = next
		if buf.empty() {
			return nil
		}
		return buf
	}

	switch line.Role {
	case "user":
		if buf.userText == "" {
			buf.userText = line.Text
		} else if line.Text != "" {
			buf.userText += "\n" + line.Text
		}
	case "assistant":
		if line.Text != "" {
			if buf.assistantText.Len() > 0 {
				buf.assistantText.WriteString("\n")
			}
			buf.assistantText.WriteString(line.Text)
		}
		buf.toolCalls = append(buf.toolCalls, line.ToolCalls...)
	}
	// Tool results attach regardless of the carrier line's role (Claude
	// delivers them on "user" lines; Cursor on assistant lines).
	buf.toolResults = append(buf.toolResults, line.ToolResults...)

	// Non-user turn end (Cursor turn_ended, Codex turn_context, Claude
	// end_turn): the buffer is complete including this line's content.
	if line.TurnEnded {
		a.buf = nil
		if buf.empty() {
			return nil
		}
		return buf
	}
	return nil
}

// Idle returns the pending buffer if it has content and has not been written
// to since `cutoff`, clearing it from the assembler. Used by the watcher's
// idle-flush timer so a session's final turn (which never gets an explicit
// end marker in Pi/Codex) is still emitted.
func (a *turnAssembler) Idle(cutoff time.Time) *turnBuffer {
	if a.buf == nil || a.buf.empty() || a.buf.lastWriteAt.After(cutoff) {
		return nil
	}
	buf := a.buf
	a.buf = nil
	return buf
}

// Drain returns the pending buffer (if non-empty) and clears it. Used on
// watcher shutdown so in-flight turns are not lost.
func (a *turnAssembler) Drain() *turnBuffer {
	if a.buf == nil || a.buf.empty() {
		a.buf = nil
		return nil
	}
	buf := a.buf
	a.buf = nil
	return buf
}
