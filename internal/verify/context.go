package verify

import (
	"encoding/json"

	"github.com/fristovic/snitch/internal/capture"
	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/severity"
	"github.com/fristovic/snitch/internal/transcript"
	"github.com/fristovic/snitch/internal/verify/verifiers"
)

const sessionLookbackTurns = 3

// BuildVerifyContext assembles enriched verification context for a run.
func BuildVerifyContext(store *record.Store, payload capture.RunPayload) (verifiers.VerifyContext, error) {
	execText, recapText := segmentProse(payload.AssistantText)

	subCalls, err := transcript.LoadSubagentToolCalls(payload.TranscriptPath, payload.StartedAt, payload.FinishedAt)
	if err != nil {
		return verifiers.VerifyContext{}, err
	}

	effective := append([]transcript.ToolCall{}, payload.ToolCalls...)
	effective = append(effective, subCalls...)

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
		PriorTurns:         priorTurns,
		ExecutionText:      execText,
		RecapText:          recapText,
		AssistantText:      payload.AssistantText,
	}, nil
}

func payloadToEvidence(p capture.RunPayload) verifiers.TurnEvidence {
	return verifiers.TurnEvidence{
		RunID:        p.RunID,
		ToolCalls:    p.ToolCalls,
		StartHEAD:    p.StartHEAD,
		EndHEAD:      p.EndHEAD,
		StartedAt:    p.StartedAt,
		FinishedAt:   p.FinishedAt,
		FileManifest: p.FileManifest,
	}
}

// AdjustSeverity calibrates claim severity using confidence and segment.
func AdjustSeverity(base severity.Level, claim verifiers.Claim, accurate bool) severity.Level {
	if accurate {
		return base
	}
	if claim.Segment == "recap" {
		if base >= severity.Level3 {
			return severity.Level2
		}
		return base
	}
	if claim.Confidence <= 1 && claim.Source == "prose" && base > severity.Level1 {
		switch claim.Type {
		case verifiers.ClaimNoAction, verifiers.ClaimCommitted, verifiers.ClaimPushed,
			verifiers.ClaimTestPass, verifiers.ClaimFileCreated, verifiers.ClaimFileModified, verifiers.ClaimFileDeleted:
			return base
		default:
			return severity.Level1
		}
	}
	return base
}

// RecapEscalateSeverity upgrades recap lies to FAIL when no session evidence exists.
func RecapEscalateSeverity(sev severity.Level, claim verifiers.Claim, accurate bool, ctx verifiers.VerifyContext) severity.Level {
	if accurate || claim.Segment != "recap" {
		return sev
	}
	if verifiers.SessionHasZeroEvidence(ctx, claim.Type) && sev < severity.Level3 {
		return severity.Level3
	}
	return sev
}
