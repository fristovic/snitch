package verifiers

import (
	"os/exec"
	"strings"
	"time"

	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/platform"
	"github.com/fristovic/snitch/internal/severity"
)

// ShellVerifier checks Shell tool calls.
type ShellVerifier struct {
	AllowRerun map[string]bool
}

func NewShellVerifier(cfg config.ShellVerifierConfig) *ShellVerifier {
	return &ShellVerifier{AllowRerun: cfg.AllowRerun}
}

func (v *ShellVerifier) Name() string { return "shell" }

func (v *ShellVerifier) CanHandle(c Claim) bool {
	return c.Source == "tool" && c.Type == ClaimToolShell
}

func (v *ShellVerifier) Verify(c Claim, ctx VerifyContext) (Result, error) {
	r := Result{Claim: c, Verifier: v.Name(), Epistemic: EpistemicSupported, Severity: severity.Level0}
	cmd, _ := c.Input["command"].(string)
	if cmd == "" {
		cmd = c.Target
	}
	if strings.TrimSpace(cmd) == "" {
		r.Epistemic = EpistemicMissing
		r.Severity = severity.Level1
		r.GroundTruth = "empty shell command"
		return r, nil
	}

	if isGitCommitCommand(cmd) {
		return v.verifyGitCommitShell(cmd, ctx)
	}
	if isGitPushCommand(cmd) {
		return v.verifyGitPushShell(cmd, ctx)
	}

	for _, tc := range AllToolCalls(ctx) {
		if tc.Name != "Shell" {
			continue
		}
		if ShellCommand(tc) != cmd && tc.Target != cmd {
			continue
		}
		if eval, ok := EvaluateCapturedShellCommand(cmd, tc, ctx, severity.Level2); ok {
			eval.Claim = c
			eval.Verifier = v.Name()
			return eval, nil
		}
	}

	passed, found := parseTestOutput(ctx.Output)
	if found {
		if strings.Contains(strings.ToLower(cmd), "test") {
			if passed {
				r.GroundTruth = "test output indicates pass"
			} else {
				r.Epistemic = EpistemicContradicted
				r.Severity = severity.Level2
				r.GroundTruth = "test output indicates failure"
			}
			return r, nil
		}
	}

	cwd := ctx.Cwd
	if wd, ok := c.Input["working_directory"].(string); ok && wd != "" {
		cwd = wd
	}
	if v.AllowRerun != nil && v.AllowRerun[cwd] {
		code := rerunCommand(cmd, cwd)
		if code != 0 {
			r.Epistemic = EpistemicContradicted
			r.Severity = severity.Level2
			r.GroundTruth = "re-run exited with code " + formatSize(int64(code))
		} else {
			r.GroundTruth = "re-run succeeded"
		}
		return r, nil
	}

	r.Epistemic = EpistemicMissing
	r.Severity = severity.Level1
	r.GroundTruth = "command syntax valid (re-run not enabled)"
	return r, nil
}

func isGitCommitCommand(cmd string) bool {
	return strings.Contains(strings.ToLower(cmd), "git commit")
}

func isGitPushCommand(cmd string) bool {
	return strings.Contains(strings.ToLower(cmd), "git push")
}

func (v *ShellVerifier) verifyGitCommitShell(cmd string, ctx VerifyContext) (Result, error) {
	claim := Claim{
		Type:        ClaimCommitted,
		Source:      "tool",
		Description: cmd,
	}
	return VerifyCommitEvidence(ctx, v.Name(), claim), nil
}

func (v *ShellVerifier) verifyGitPushShell(cmd string, ctx VerifyContext) (Result, error) {
	claim := Claim{
		Type:        ClaimPushed,
		Source:      "tool",
		Description: cmd,
	}
	return VerifyPushEvidence(ctx, v.Name(), claim), nil
}

func parseTestOutput(output string) (passed bool, found bool) {
	return ParseTestOutput(output)
}

func rerunCommand(cmd, cwd string) int {
	c := platform.ShellCommand(cmd)
	c.Dir = cwd
	done := make(chan error, 1)
	go func() { done <- c.Run() }()
	select {
	case err := <-done:
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return exitErr.ExitCode()
			}
			return 1
		}
		return 0
	case <-time.After(30 * time.Second):
		if c.Process != nil {
			_ = c.Process.Kill()
		}
		return 1
	}
}
