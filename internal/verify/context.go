package verify

import (
	"encoding/json"
	"log/slog"

	"github.com/fristovic/snitch/internal/capture"
	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/severity"
	"github.com/fristovic/snitch/internal/transcript"
	"github.com/fristovic/snitch/internal/verify/verifiers"
)

const sessionLookbackTurns = 3

// MergedToolCalls returns parent tool calls plus subagent evidence for a turn.
func MergedToolCalls(p capture.RunPayload) ([]transcript.ToolCall, error) {
	effective := append([]transcript.ToolCall{}, p.ToolCalls...)
	if p.TranscriptPath == "" {
		return effective, nil
	}
	sub, err := transcript.LoadSubagentToolCalls(p.TranscriptPath, p.StartedAt, p.FinishedAt)
	if err != nil {
		return effective, err
	}
	return append(effective, sub...), nil
}

func baseVerifyContext(payload capture.RunPayload) verifiers.VerifyContext {
	effective, _ := MergedToolCalls(payload)
	execText, recapText := segmentProse(payload.AssistantText)
	endHEAD := payload.EndHEAD
	if endHEAD == "" && payload.ProjectPath != "" {
		endHEAD = verifiers.GitHEADAt(payload.ProjectPath)
	}
	return verifiers.VerifyContext{
		Output:             payload.Output,
		Cwd:                payload.ProjectPath,
		ProjectPath:        payload.ProjectPath,
		StartHEAD:          payload.StartHEAD,
		EndHEAD:            endHEAD,
		FileManifest:       payload.FileManifest,
		TranscriptPath:     payload.TranscriptPath,
		ObservedAt:         payload.FinishedAt,
		StartedAt:          payload.StartedAt,
		FinishedAt:         payload.FinishedAt,
		ToolCalls:          payload.ToolCalls,
		EffectiveToolCalls: effective,
		ExecutionText:      execText,
		RecapText:          recapText,
		AssistantText:      payload.AssistantText,
	}
}

// BuildVerifyContext assembles enriched verification context for a run.
func BuildVerifyContext(store *record.Store, payload capture.RunPayload) (verifiers.VerifyContext, error) {
	effective, err := MergedToolCalls(payload)
	if err != nil {
		slog.Debug("subagent tool calls unavailable", "err", err)
	}

	var priorTurns []verifiers.TurnEvidence
	if store != nil && payload.SessionID != "" {
		raws, err := store.GetPriorRunPayloadJSON(payload.SessionID, payload.StartedAt, sessionLookbackTurns)
		if err != nil {
			return verifiers.VerifyContext{}, err
		}
		for _, raw := range raws {
			var p capture.RunPayload
			if err := json.Unmarshal(raw, &p); err != nil {
				continue
			}
			priorTurns = append(priorTurns, payloadToEvidence(p))
		}
	}

	vctx := baseVerifyContext(payload)
	vctx.EffectiveToolCalls = effective
	vctx.PriorTurns = priorTurns
	return vctx, nil
}

func payloadToEvidence(p capture.RunPayload) verifiers.TurnEvidence {
	calls, _ := MergedToolCalls(p)
	return verifiers.TurnEvidence{
		RunID:        p.RunID,
		ToolCalls:    calls,
		StartHEAD:    p.StartHEAD,
		EndHEAD:      p.EndHEAD,
		StartedAt:    p.StartedAt,
		FinishedAt:   p.FinishedAt,
		FileManifest: p.FileManifest,
	}
}

// ApplyClaimPolicy calibrates verifier severity using confidence, segment, and session evidence.
func ApplyClaimPolicy(sev severity.Level, claim verifiers.Claim, accurate bool, ctx verifiers.VerifyContext) severity.Level {
	if accurate {
		return sev
	}
	if claim.Segment == "recap" {
		if sev >= severity.Level3 {
			sev = severity.Level2
		}
	} else if claim.Confidence <= 1 && claim.Source == "prose" && sev > severity.Level1 {
		switch claim.Type {
		case verifiers.ClaimNoAction, verifiers.ClaimCommitted, verifiers.ClaimPushed,
			verifiers.ClaimTestPass, verifiers.ClaimFileCreated, verifiers.ClaimFileModified, verifiers.ClaimFileDeleted:
		default:
			sev = severity.Level1
		}
	}
	if claim.Segment == "recap" && verifiers.SessionHasZeroEvidence(ctx, claim.Type) && sev < severity.Level3 {
		return severity.Level3
	}
	return sev
}
