// Package stress provides table-driven claim-verifier stress tests.
package stress

import (
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/fristovic/snitch/internal/capture"
	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/transcript"
	"github.com/fristovic/snitch/internal/verify"
)

// Category classifies expected detector behavior for a case.
type Category string

const (
	CategoryFalsePositive Category = "false_positive"
	CategoryFalseNegative Category = "false_negative"
	CategoryTruePositive  Category = "true_positive"
	CategoryTrueNegative  Category = "true_negative"
)

// StressCase is one prose + tool-call scenario with expected flagged-claim outcome.
type StressCase struct {
	Name            string
	ClaimType       string
	Category        Category
	AssistantText   string
	ToolCalls       []transcript.ToolCall
	ProjectFiles    map[string]string
	StartHEAD       string
	EndHEAD         string
	SessionID       string
	ExpectFlagged   bool
	ExpectClaimType string
	Notes           string
}

// CaseResult is the observed verification outcome.
type CaseResult struct {
	Verdict    record.Verdict
	Claims     []record.Claim
	MatchedFlagged bool
	MatchedClaim   *record.Claim
}

// RunCase executes verification and returns observed outcomes.
func RunCase(sc StressCase, dataDir, projectDir string) (CaseResult, error) {
	store, err := record.Open(dataDir)
	if err != nil {
		return CaseResult{}, err
	}
	defer store.Close()

	deviceID, err := store.EnsureDeviceID()
	if err != nil {
		return CaseResult{}, err
	}

	for rel, body := range sc.ProjectFiles {
		abs := filepath.Join(projectDir, rel)
		if err := writeFile(abs, body); err != nil {
			return CaseResult{}, err
		}
	}
	if err := applyToolCalls(projectDir, sc.ToolCalls); err != nil {
		return CaseResult{}, err
	}

	now := time.Now()
	return runOneTurn(store, deviceID, sc, projectDir, now.Add(-time.Minute), now)
}

// RunSession executes turns in order sharing a session for lookback tests.
func RunSession(cases []StressCase, dataDir, projectDir, sessionID string) ([]CaseResult, error) {
	store, err := record.Open(dataDir)
	if err != nil {
		return nil, err
	}
	defer store.Close()

	deviceID, err := store.EnsureDeviceID()
	if err != nil {
		return nil, err
	}

	var results []CaseResult
	base := time.Now().Add(-time.Hour)
	for i, sc := range cases {
		for rel, body := range sc.ProjectFiles {
			abs := filepath.Join(projectDir, rel)
			if err := writeFile(abs, body); err != nil {
				return nil, err
			}
		}
		if err := applyToolCalls(projectDir, sc.ToolCalls); err != nil {
			return nil, err
		}

		started := base.Add(time.Duration(i) * time.Minute)
		finished := started.Add(30 * time.Second)
		res, err := runOneTurn(store, deviceID, sc, projectDir, started, finished)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return results, nil
}

func runOneTurn(store *record.Store, deviceID string, sc StressCase, projectDir string, started, finished time.Time) (CaseResult, error) {
	runID := "stress-" + sc.Name
	sessionID := sc.SessionID
	if sessionID == "" {
		sessionID = "stress-session"
	}
	payload := capture.RunPayload{
		RunID:         runID,
		SessionID:     sessionID,
		ProjectPath:   projectDir,
		AssistantText: sc.AssistantText,
		Output:        sc.AssistantText,
		ToolCalls:     sc.ToolCalls,
		ToolCallCount: len(sc.ToolCalls),
		Harness:       "cursor",
		StartHEAD:     sc.StartHEAD,
		EndHEAD:       sc.EndHEAD,
		FileManifest:  transcript.BuildFileManifest(projectDir, sc.ToolCalls),
		StartedAt:     started,
		FinishedAt:    finished,
	}

	bus := newTestBus()
	engine := verify.NewEngine(bus, store, config.Default().Verification, deviceID, nil)
	engine.VerifyPayload(payload)

	run, err := store.GetRunByID(runID)
	if err != nil {
		return CaseResult{}, err
	}
	claims, err := store.GetClaimsByRun(runID)
	if err != nil {
		return CaseResult{}, err
	}

	res := CaseResult{Claims: claims}
	if run != nil {
		res.Verdict = run.Verdict
	}
	claimType := sc.ExpectClaimType
	if claimType == "" {
		claimType = sc.ClaimType
	}
	for i := range claims {
		c := &claims[i]
		if c.ClaimType != claimType {
			continue
		}
		if record.IsContradictedClaim(*c) && c.Severity >= 2 {
			res.MatchedFlagged = true
			cp := *c
			res.MatchedClaim = &cp
			break
		}
	}
	return res, nil
}

// AllCases returns every stress case across families.
func AllCases() []StressCase {
	var out []StressCase
	out = append(out, LiveSessionCases()...)
	out = append(out, FileProseCases()...)
	out = append(out, StubNoActionCases()...)
	out = append(out, GitCases()...)
	out = append(out, ShellCases()...)
	out = append(out, ConsistencyCases()...)
	return out
}

func strReplace(path, old, newStr string) transcript.ToolCall {
	return transcript.ToolCall{
		Name:   "StrReplace",
		Target: path,
		Input: map[string]json.RawMessage{
			"path":       mustRaw(path),
			"old_string": mustRaw(old),
			"new_string": mustRaw(newStr),
		},
	}
}

func write(path, contents string) transcript.ToolCall {
	return transcript.ToolCall{
		Name:   "Write",
		Target: path,
		Input: map[string]json.RawMessage{
			"path":     mustRaw(path),
			"contents": mustRaw(contents),
		},
	}
}

func del(path string) transcript.ToolCall {
	return transcript.ToolCall{
		Name:   "Delete",
		Target: path,
		Input: map[string]json.RawMessage{
			"path": mustRaw(path),
		},
	}
}

func shell(cmd string, result string, isError bool) transcript.ToolCall {
	return transcript.ToolCall{
		Name:      "Shell",
		Target:    cmd,
		Result:    result,
		IsError:   isError,
		ToolUseID: "shell-" + cmd,
		Input: map[string]json.RawMessage{
			"command": mustRaw(cmd),
		},
	}
}

func read(path string) transcript.ToolCall {
	return transcript.ToolCall{
		Name:   "Read",
		Target: path,
		Input: map[string]json.RawMessage{
			"path": mustRaw(path),
		},
	}
}

func mustRaw(s string) json.RawMessage {
	b, _ := json.Marshal(s)
	return b
}
