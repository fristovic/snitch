package verifiers

import (
	"os/exec"
	"strings"

	"github.com/fristovic/snitch/internal/transcript"
)

// HasCommitEvidence reports git commit evidence in current or prior turns.
func HasCommitEvidence(ctx VerifyContext) bool {
	if HasGitCommitShell(AllToolCalls(ctx)) {
		return true
	}
	cwd := ctx.ProjectPath
	if cwd == "" {
		cwd = ctx.Cwd
	}
	if hasNewCommit(cwd, ctx.StartHEAD) && HasGitCommitShell(AllToolCalls(ctx)) {
		return true
	}
	for _, pt := range ctx.PriorTurns {
		if HasGitCommitShell(pt.ToolCalls) {
			return true
		}
		if hasNewCommit(cwd, pt.StartHEAD) && pt.EndHEAD != "" && pt.EndHEAD != pt.StartHEAD {
			return true
		}
	}
	return false
}

// HasPushEvidence reports git push evidence in current or prior turns.
func HasPushEvidence(ctx VerifyContext) bool {
	if HasGitPushShell(AllToolCalls(ctx)) {
		return true
	}
	for _, pt := range ctx.PriorTurns {
		if HasGitPushShell(pt.ToolCalls) {
			return true
		}
	}
	return false
}

// HasTestShellEvidence reports test shell commands in current or prior turns.
func HasTestShellEvidence(ctx VerifyContext) bool {
	if HasTestShell(AllToolCalls(ctx)) {
		return true
	}
	for _, pt := range ctx.PriorTurns {
		if HasTestShell(pt.ToolCalls) {
			return true
		}
	}
	return false
}

// HasShellEvidence reports any shell tool call in current or prior turns.
func HasShellEvidence(ctx VerifyContext) bool {
	if HasShellCall(AllToolCalls(ctx)) {
		return true
	}
	for _, pt := range ctx.PriorTurns {
		if HasShellCall(pt.ToolCalls) {
			return true
		}
	}
	return false
}

// HasFileToolForPath reports file mutation tools targeting path in current or prior turns.
func HasFileToolForPath(ctx VerifyContext, target string) bool {
	cwd := ctx.ProjectPath
	if cwd == "" {
		cwd = ctx.Cwd
	}
	abs := resolveClaimPath(target, cwd)
	if hasAnyFileToolForPath(AllToolCalls(ctx), target, abs, cwd) {
		return true
	}
	for _, pt := range ctx.PriorTurns {
		if hasAnyFileToolForPath(pt.ToolCalls, target, abs, cwd) {
			return true
		}
	}
	return false
}

func hasAnyFileToolForPath(calls []transcript.ToolCall, target, absPath, cwd string) bool {
	tools := []string{"Write", "StrReplace", "Delete"}
	for _, tool := range tools {
		if hasToolForPath(calls, tool, target, absPath, cwd, tool == "Delete") {
			return true
		}
	}
	return false
}

// SessionHasZeroEvidence reports whether no checkable evidence exists across lookback.
func SessionHasZeroEvidence(ctx VerifyContext, claimType ClaimType) bool {
	switch claimType {
	case ClaimCommitted:
		return !HasCommitEvidence(ctx)
	case ClaimPushed:
		return !HasPushEvidence(ctx)
	case ClaimTestPass:
		return !HasTestShellEvidence(ctx)
	case ClaimCommandRan, ClaimCommandSucceeded:
		return !HasShellEvidence(ctx)
	case ClaimFileCreated, ClaimFileModified, ClaimFileDeleted:
		return false
	default:
		return false
	}
}

// GitHEADAt returns current HEAD for project.
func GitHEADAt(cwd string) string {
	return transcript.GitHEAD(cwd)
}

func hasNewCommit(cwd, startHEAD string) bool {
	if cwd == "" || startHEAD == "" {
		return false
	}
	current, err := exec.Command("git", "-C", cwd, "rev-parse", "HEAD").Output()
	if err != nil {
		return false
	}
	cur := strings.TrimSpace(string(current))
	if cur == "" || cur == startHEAD {
		return false
	}
	out, err := exec.Command("git", "-C", cwd, "rev-list", "--count", startHEAD+"..HEAD").Output()
	if err != nil {
		return cur != startHEAD
	}
	n := strings.TrimSpace(string(out))
	return n != "0" && n != ""
}
