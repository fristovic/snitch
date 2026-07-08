package transcript

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ReplayTurns parses a whole transcript file offline and assembles it into
// completed turns using the same turnAssembler the live watcher uses. The
// trailing buffer is drained so the final turn (which may lack an explicit
// end marker) is included. Used by `snitch replay` and contributor tooling.
func ReplayTurns(parser TranscriptParser, resolver PathResolver, path string) ([]TurnCompleted, error) {
	lines, _, err := ParseLinesWith(parser, path, 0)
	if err != nil {
		return nil, err
	}
	a := newTurnAssembler(resolver, path)
	var bufs []*turnBuffer
	for _, line := range lines {
		if buf := a.Feed(line); buf != nil {
			bufs = append(bufs, buf)
		}
	}
	if buf := a.Drain(); buf != nil {
		bufs = append(bufs, buf)
	}

	turns := make([]TurnCompleted, 0, len(bufs))
	for i, buf := range bufs {
		projectPath := buf.projectPath
		if projectPath == "" {
			projectPath = resolver.ProjectCwd(path)
		}
		toolCalls := AttachToolResults(buf.toolCalls, buf.toolResults)
		turns = append(turns, TurnCompleted{
			RunID:          fmt.Sprintf("replay-%s-%d", strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), i+1),
			SessionID:      resolver.SessionID(path),
			TranscriptPath: path,
			ProjectPath:    projectPath,
			Model:          buf.model,
			StartHEAD:      buf.startHEAD,
			UserText:       buf.userText,
			AssistantText:  buf.assistantText.String(),
			ToolCalls:      toolCalls,
			StartedAt:      buf.startedAt,
			FinishedAt:     buf.lastWriteAt,
		})
	}
	return turns, nil
}

// GuessHarness infers the harness from a transcript path, or "" if unknown.
func GuessHarness(path string) string {
	p := filepath.ToSlash(path)
	switch {
	case strings.Contains(p, ".cursor/") || strings.Contains(p, "agent-transcripts"):
		return "cursor"
	case strings.Contains(p, ".claude/"):
		return "claude"
	case strings.Contains(p, ".codex/") || strings.HasPrefix(filepath.Base(p), "rollout-"):
		return "codex"
	case strings.Contains(p, ".pi/"):
		return "pi"
	}
	return ""
}

// ParserFor returns the JSONL parser + resolver for a harness name, or ok=false
// for unknown or non-JSONL harnesses (OpenCode replays are not file-based).
func ParserFor(harness string) (TranscriptParser, PathResolver, bool) {
	switch harness {
	case "cursor":
		return CursorParser{}, CursorPathResolver{}, true
	case "claude":
		return ClaudeParser{}, ClaudePathResolver{}, true
	case "codex":
		return CodexParser{}, CodexPathResolver{}, true
	case "pi":
		return PiParser{}, PiPathResolver{}, true
	}
	return nil, nil, false
}
