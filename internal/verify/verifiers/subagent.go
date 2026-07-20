package verifiers

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/fristovic/snitch/internal/severity"
	"github.com/fristovic/snitch/internal/transcript"
)

// SubagentVerifier checks Task tool calls spawn subagent transcripts.
type SubagentVerifier struct{}

func (v *SubagentVerifier) Name() string { return "subagent" }

func (v *SubagentVerifier) CanHandle(c Claim) bool {
	return c.Source == "tool" && c.Type == ClaimToolTask
}

func (v *SubagentVerifier) Verify(c Claim, ctx VerifyContext) (Result, error) {
	r := Result{Claim: c, Verifier: v.Name(), Epistemic: EpistemicSupported, Severity: severity.Level0}
	if ctx.TranscriptPath == "" {
		r.Epistemic = EpistemicMissing
		r.Severity = severity.Level2
		r.GroundTruth = "no parent transcript path"
		return r, nil
	}

	subDir := filepath.Join(filepath.Dir(ctx.TranscriptPath), "subagents")
	entries, err := os.ReadDir(subDir)
	if err != nil {
		r.Epistemic = EpistemicMissing
		r.Severity = severity.Level2
		r.GroundTruth = "subagents directory missing"
		return r, nil
	}

	taskCount := 0
	for _, tc := range ctx.ToolCalls {
		if tc.Name == "Task" {
			taskCount++
		}
	}
	if taskCount == 0 {
		taskCount = 1
	}

	nonEmpty := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.Size() > 0 {
			nonEmpty++
		}
	}

	if nonEmpty == 0 {
		r.Epistemic = EpistemicContradicted
		r.Severity = severity.Level1
		r.GroundTruth = "subagent transcripts empty"
		return r, nil
	}
	if nonEmpty < taskCount {
		r.Epistemic = EpistemicContradicted
		r.Severity = severity.Level2
		r.GroundTruth = "fewer subagent transcripts than Task calls"
		return r, nil
	}

	merged, _ := transcript.LoadSubagentToolCalls(ctx.TranscriptPath, ctx.StartedAt, ctx.FinishedAt)
	if taskCount > 0 && len(merged) == 0 && !ctx.StartedAt.IsZero() && !ctx.FinishedAt.IsZero() {
		r.Epistemic = EpistemicMissing
		r.Severity = severity.Level1
		r.GroundTruth = "subagent transcripts exist but no tool calls in turn window"
		return r, nil
	}

	harness := ctx.Harness
	if harness == "" {
		harness = transcript.GuessHarness(ctx.TranscriptPath)
	}
	_, resolver, ok := transcript.ParserFor(harness)
	if !ok {
		resolver = transcript.CursorPathResolver{}
	}
	r.GroundTruth = resolver.SessionID(ctx.TranscriptPath) + ": " +
		formatSize(int64(nonEmpty)) + " subagent transcript(s)"
	return r, nil
}
