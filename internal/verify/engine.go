package verify

import (
	"encoding/json"
	"log/slog"
	"regexp"

	"github.com/fristovic/snitch/internal/capture"
	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/severity"
	"github.com/fristovic/snitch/internal/transcript"
	"github.com/fristovic/snitch/internal/verify/verifiers"
)

var reGenericActionProse = regexp.MustCompile(`(?i)\b(?:i\s+)?(?:modified|updated|created|committed|pushed|deleted|removed|wrote|added)\b`)

// Engine runs the lie-detection verification pipeline.
type Engine struct {
	bus        *event.Bus
	store      *record.Store
	cfg        config.VerificationConfig
	verifiers  []verifiers.Verifier
	deviceID   string
	sem        chan struct{}
	onVerified func(event.RunVerifiedPayload)
}

// NewEngine creates a verification engine.
func NewEngine(bus *event.Bus, store *record.Store, cfg config.VerificationConfig, deviceID string) *Engine {
	maxConc := cfg.MaxConcurrentVerifications
	if maxConc <= 0 {
		maxConc = 3
	}
	return &Engine{
		bus:   bus,
		store: store,
		cfg:   cfg,
		verifiers: []verifiers.Verifier{
			&verifiers.ContradictionVerifier{},
			&verifiers.ConsistencyVerifier{},
			&verifiers.FileVerifier{},
			verifiers.NewShellVerifier(cfg.ShellVerifier),
			&verifiers.SubagentVerifier{},
		},
		deviceID: deviceID,
		sem:      make(chan struct{}, maxConc),
	}
}

// OnVerified registers a callback invoked after each verified run.
func (e *Engine) OnVerified(fn func(event.RunVerifiedPayload)) {
	e.onVerified = fn
}

// Start listens for RunCaptured events.
func (e *Engine) Start() {
	ch := e.bus.Subscribe(event.EventRunCaptured)
	go func() {
		for ev := range ch {
			e.sem <- struct{}{}
			go func(ev event.Event) {
				defer func() { <-e.sem }()
				e.process(ev)
			}(ev)
		}
	}()
}

// VerifyPayload runs the verification pipeline synchronously (for tests).
func (e *Engine) VerifyPayload(payload capture.RunPayload) {
	e.process(event.Event{
		ID:        payload.RunID,
		Timestamp: payload.FinishedAt,
		Source:    "test",
		Type:      event.EventRunCaptured,
		Payload:   mustJSON(payload),
	})
}

func mustJSON(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func (e *Engine) process(ev event.Event) {
	var payload capture.RunPayload
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		slog.Warn("invalid run payload", "err", err)
		return
	}

	hash := capture.HashOutput(payload.Output)
	if hash != "" {
		exists, err := e.store.RunExistsByOutputHash(hash)
		if err != nil {
			slog.Warn("dedup check failed", "err", err)
		} else if exists {
			slog.Debug("duplicate run skipped", "hash", hash)
			return
		}
	}

	run := record.Run{
		ID:             payload.RunID,
		SessionID:      payload.SessionID,
		TranscriptPath: payload.TranscriptPath,
		ProjectPath:    payload.ProjectPath,
		Command:        payload.Command,
		Harness:        payload.Harness,
		OutputHash:     hash,
		DeviceID:       e.deviceID,
		DurationMS:     payload.FinishedAt.Sub(payload.StartedAt).Milliseconds(),
		ToolCallCount:  payload.ToolCallCount,
		CreatedAt:      payload.FinishedAt.UTC(),
	}
	if err := e.store.InsertRun(run); err != nil {
		slog.Error("insert run failed", "run_id", payload.RunID, "err", err)
		return
	}

	vctx, err := BuildVerifyContext(e.store, payload)
	if err != nil {
		slog.Warn("build verify context failed", "err", err)
		vctx = verifiers.VerifyContext{
			Output: payload.Output, Cwd: payload.ProjectPath, ProjectPath: payload.ProjectPath,
			StartHEAD: payload.StartHEAD, EndHEAD: payload.EndHEAD, FileManifest: payload.FileManifest,
			TranscriptPath: payload.TranscriptPath, ObservedAt: payload.FinishedAt,
			StartedAt: payload.StartedAt, FinishedAt: payload.FinishedAt,
			ToolCalls: payload.ToolCalls, EffectiveToolCalls: payload.ToolCalls,
			AssistantText: payload.AssistantText,
		}
	}

	claims := e.buildClaims(payload, vctx)
	trace := []string{"Phase 1: Extract prose + tool claims", "claims=" + itoa(len(claims))}

	maxSev := severity.Level0
	verified, falseClaims := 0, 0

	for _, claim := range claims {
		best := verifiers.Result{Claim: claim, Severity: severity.Level(-1)}
		for _, v := range e.verifiers {
			if !v.CanHandle(claim) {
				continue
			}
			res, err := v.Verify(claim, vctx)
			if err != nil {
				slog.Debug("verifier error", "verifier", v.Name(), "err", err)
				continue
			}
			if res.Verifier == "" {
				res.Verifier = v.Name()
			}
			if best.Verifier == "" || res.Severity > best.Severity || (res.Accurate && !best.Accurate) {
				best = res
			}
		}
		if best.Verifier == "" {
			continue
		}
		best.Severity = AdjustSeverity(best.Severity, claim, best.Accurate)
		best.Severity = RecapEscalateSeverity(best.Severity, claim, best.Accurate, vctx)
		if best.Accurate {
			verified++
		} else if best.Severity >= severity.Level2 {
			falseClaims++
		}
		if best.Severity > maxSev {
			maxSev = best.Severity
		}
		claimed := claim.Quote
		if claimed == "" {
			claimed = claim.Description
		}
		recClaim := record.Claim{
			RunID:      payload.RunID,
			ClaimType:  string(claim.Type),
			Source:     claim.Source,
			Target:     claim.Target,
			Claimed:    claimed,
			Actual:     best.GroundTruth,
			Verified:   boolToVerified(best.Accurate),
			Severity:   int(best.Severity),
			Verifier:   best.Verifier,
			Evidence:   best.Evidence,
			Confidence: claim.Confidence,
		}
		if err := e.store.InsertClaims([]record.Claim{recClaim}); err != nil {
			slog.Error("insert claim failed", "err", err)
		}
	}

	verdict := record.Verdict(severity.Verdict(maxSev))
	trace = append(trace, "Phase 2: Verdict → "+string(verdict))
	if err := e.store.UpdateRunVerdict(payload.RunID, verdict, int(maxSev), len(claims), verified, falseClaims); err != nil {
		slog.Error("update verdict failed", "err", err)
	}
	if err := e.store.SaveTrace(payload.RunID, trace); err != nil {
		slog.Warn("save trace failed", "err", err)
	}

	endHEAD := payload.EndHEAD
	if endHEAD == "" {
		endHEAD = vctx.EndHEAD
	}
	payloadBytes, _ := json.Marshal(payload)
	if err := e.store.SaveTurnSnapshot(payload.RunID, payloadBytes, payload.StartHEAD, endHEAD, payload.FileManifest); err != nil {
		slog.Warn("save turn snapshot failed", "err", err)
	}

	e.emitVerified(event.RunVerifiedPayload{
		RunID:       payload.RunID,
		Verdict:     verdict,
		MaxSeverity: int(maxSev),
		Command:     payload.Command,
		ProjectPath: payload.ProjectPath,
		SessionID:   payload.SessionID,
	})
}

func (e *Engine) buildClaims(payload capture.RunPayload, vctx verifiers.VerifyContext) []verifiers.Claim {
	claims := ExtractProseClaims(payload.AssistantText)
	claims = filterFileClaimsWithGitCommitOnly(claims, verifiers.AllToolCalls(vctx), vctx)
	claims = append(claims, verifiers.ExtractConsistencyClaims(payload.AssistantText, payload.ToolCalls)...)
	if !hasMutatingToolCalls(payload.ToolCalls) && (HasLocalActionProse(claims) || reGenericActionProse.MatchString(payload.AssistantText)) {
		claims = append(claims, verifiers.Claim{
			Type:        verifiers.ClaimNoAction,
			Source:      "prose",
			Description: "action claimed in prose with zero tool calls",
		})
	}
	for _, tc := range payload.ToolCalls {
		if claim, ok := verifiers.ToolCallToClaim(tc); ok {
			claims = append(claims, claim)
		}
	}
	return claims
}

func hasMutatingToolCalls(calls []transcript.ToolCall) bool {
	readOnly := map[string]bool{
		"Read": true, "Glob": true, "Grep": true, "SemanticSearch": true,
	}
	for _, tc := range calls {
		if !readOnly[tc.Name] {
			return true
		}
	}
	return false
}

func (e *Engine) emitVerified(p event.RunVerifiedPayload) {
	data, _ := json.Marshal(p)
	e.bus.Publish(event.Event{
		ID: p.RunID, Timestamp: capture.NowUTC(), Source: "verify",
		Type: event.EventRunVerified, Payload: data,
	})
	if e.onVerified != nil {
		e.onVerified(p)
	}
}

func boolToVerified(ok bool) int {
	if ok {
		return 1
	}
	return -1
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
