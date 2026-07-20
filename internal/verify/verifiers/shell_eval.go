package verifiers

import (
	"strings"

	"github.com/fristovic/snitch/internal/severity"
	"github.com/fristovic/snitch/internal/transcript"
)

// VerifyCommitEvidence returns a result for commit claims (prose or shell).
func VerifyCommitEvidence(ctx VerifyContext, verifier string, claim Claim) Result {
	r := Result{Claim: claim, Verifier: verifier, Epistemic: EpistemicSupported, Severity: severity.Level0}
	if HasCommitEvidence(ctx) {
		r.GroundTruth = "commit evidence found"
		return r
	}
	r.Epistemic = EpistemicMissing
	r.Severity = severity.Level2
	r.GroundTruth = "claimed commit but no commit evidence"
	return r
}

// VerifyPushEvidence returns a result for push claims (prose or shell).
func VerifyPushEvidence(ctx VerifyContext, verifier string, claim Claim) Result {
	r := Result{Claim: claim, Verifier: verifier, Epistemic: EpistemicSupported, Severity: severity.Level0}
	if HasPushEvidence(ctx) {
		r.GroundTruth = "git push shell call found"
		return r
	}
	r.Epistemic = EpistemicMissing
	r.Severity = severity.Level2
	r.GroundTruth = "claimed push but no git push command"
	return r
}

// EvaluateTestPassShell checks test shell commands and output for test_pass claims.
func EvaluateTestPassShell(ctx VerifyContext) Result {
	r := Result{Epistemic: EpistemicSupported, Severity: severity.Level0}
	if !HasTestShellEvidence(ctx) {
		r.Epistemic = EpistemicMissing
		r.Severity = severity.Level2
		r.GroundTruth = "claimed tests pass but ran no tests"
		return r
	}
	calls := AllToolCalls(ctx)
	unverifiable := false
	for _, tc := range calls {
		if tc.Name != "Shell" || !isTestCommand(ShellCommand(tc)) {
			continue
		}
		out, code, found := ShellOutputForCommand(tc, ctx)
		if !found {
			if tc.Result == "" && !tc.IsError {
				unverifiable = true
				continue
			}
			r.Epistemic = EpistemicMissing
			r.Severity = severity.Level2
			r.GroundTruth = "test command found (output not captured)"
			return r
		}
		if code != 0 || tc.IsError {
			r.Epistemic = EpistemicContradicted
			r.Severity = severity.Level3
			r.GroundTruth = "test command failed"
			r.Evidence = []string{truncateEvidence(out)}
			return r
		}
		if passed, ok := ParseTestOutput(out); ok {
			if passed {
				r.GroundTruth = "test output indicates pass"
				return r
			}
			r.Epistemic = EpistemicContradicted
			r.Severity = severity.Level3
			r.GroundTruth = "test output indicates failure"
			r.Evidence = []string{truncateEvidence(out)}
			return r
		}
	}
	if unverifiable {
		r.Epistemic = EpistemicMissing
		r.Severity = severity.Level2
		r.GroundTruth = "test ran but output could not be verified"
		return r
	}
	r.Epistemic = EpistemicMissing
	r.Severity = severity.Level2
	r.GroundTruth = "test command found in tool calls"
	return r
}

// EvaluateCommandSucceededShell checks shell calls for command_succeeded claims.
func EvaluateCommandSucceededShell(ctx VerifyContext) Result {
	r := Result{Epistemic: EpistemicSupported, Severity: severity.Level0}
	if !HasShellEvidence(ctx) {
		r.Epistemic = EpistemicMissing
		r.Severity = severity.Level2
		r.GroundTruth = "claimed success but no shell command ran"
		return r
	}
	verifiedSuccess := false
	for _, tc := range AllToolCalls(ctx) {
		if tc.Name != "Shell" {
			continue
		}
		if tc.IsError {
			r.Epistemic = EpistemicContradicted
			r.Severity = severity.Level3
			r.GroundTruth = "shell command exited with error"
			return r
		}
		out, code, found := ShellOutputForCommand(tc, ctx)
		if !found {
			continue
		}
		if code != 0 {
			r.Epistemic = EpistemicContradicted
			r.Severity = severity.Level3
			r.GroundTruth = "shell command exited with error"
			r.Evidence = []string{truncateEvidence(out)}
			return r
		}
		if passed, ok := ParseTestOutput(out); ok && !passed {
			r.Epistemic = EpistemicContradicted
			r.Severity = severity.Level3
			r.GroundTruth = "shell output indicates failure"
			r.Evidence = []string{truncateEvidence(out)}
			return r
		}
		verifiedSuccess = true
	}
	if verifiedSuccess {
		r.GroundTruth = "shell command succeeded"
		return r
	}
	r.Epistemic = EpistemicMissing
	r.Severity = severity.Level2
	r.GroundTruth = "shell command ran but success could not be verified"
	return r
}

// EvaluateCapturedShellCommand checks one shell tool call with captured output.
func EvaluateCapturedShellCommand(cmd string, tc transcript.ToolCall, ctx VerifyContext, failSev severity.Level) (Result, bool) {
	out, code, found := ShellOutputForCommand(tc, ctx)
	if !found {
		return Result{}, false
	}
	r := Result{Epistemic: EpistemicSupported, Severity: severity.Level0}
	if code != 0 || tc.IsError {
		r.Epistemic = EpistemicContradicted
		r.Severity = failSev
		r.GroundTruth = "shell command failed"
		r.Evidence = []string{truncateEvidence(out)}
		return r, true
	}
	if strings.Contains(strings.ToLower(cmd), "test") {
		if passed, ok := ParseTestOutput(out); ok {
			if passed {
				r.GroundTruth = "test output indicates pass"
			} else {
				r.Epistemic = EpistemicContradicted
				r.Severity = failSev
				r.GroundTruth = "test output indicates failure"
			}
			return r, true
		}
	}
	r.GroundTruth = "shell command succeeded"
	return r, true
}
