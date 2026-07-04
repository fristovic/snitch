package verifiers

import (
	"os/exec"
	"strings"

	"github.com/fristovic/snitch/internal/transcript"
)

// HasCommitEvidence reports git commit evidence in current or prior turns.
func HasCommitEvidence(ctx VerifyContext) bool {
	if hasGitCommitShell(AllToolCalls(ctx)) {
		return true
	}
	cwd := ctx.ProjectPath
	if cwd == "" {
		cwd = ctx.Cwd
	}
	if hasNewCommit(cwd, ctx.StartHEAD) && hasGitActivityShell(AllToolCalls(ctx)) {
		return true
	}
	for _, pt := range ctx.PriorTurns {
		if hasGitCommitShell(pt.ToolCalls) {
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
	if hasGitPushShell(AllToolCalls(ctx)) {
		return true
	}
	for _, pt := range ctx.PriorTurns {
		if hasGitPushShell(pt.ToolCalls) {
			return true
		}
	}
	return false
}

// HasTestShellEvidence reports test shell commands in current or prior turns.
func HasTestShellEvidence(ctx VerifyContext) bool {
	if hasTestShell(AllToolCalls(ctx)) {
		return true
	}
	for _, pt := range ctx.PriorTurns {
		if hasTestShell(pt.ToolCalls) {
			return true
		}
	}
	return false
}

// HasShellEvidence reports any shell tool call in current or prior turns.
func HasShellEvidence(ctx VerifyContext) bool {
	if hasShellCall(AllToolCalls(ctx)) {
		return true
	}
	for _, pt := range ctx.PriorTurns {
		if hasShellCall(pt.ToolCalls) {
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

// HasWrittenStubInTurn reports stub bodies in files written this turn or prior turns.
func HasWrittenStubInTurn(ctx VerifyContext) bool {
	cwd := ctx.ProjectPath
	if cwd == "" {
		cwd = ctx.Cwd
	}
	for _, tc := range AllToolCalls(ctx) {
		if tc.Name != "Write" && tc.Name != "StrReplace" {
			continue
		}
		if stubFromTool(tc, cwd) {
			return true
		}
	}
	for _, pt := range ctx.PriorTurns {
		for _, tc := range pt.ToolCalls {
			if tc.Name != "Write" && tc.Name != "StrReplace" {
				continue
			}
			if stubFromTool(tc, cwd) {
				return true
			}
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

func stubFromTool(tc transcript.ToolCall, cwd string) bool {
	body := writeBodyFromTool(tc)
	if body == "" {
		return false
	}
	body = stripMarkdownCodeBlocks(body)
	path := tc.Target
	if path == "" {
		path = pathFromInput(tc, tc.Name)
	}
	abs := resolveClaimPath(path, cwd)
	return IsStubBody(body, abs)
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

// PriorEndHEAD returns the most recent prior turn end HEAD.
func PriorEndHEAD(ctx VerifyContext) string {
	if len(ctx.PriorTurns) == 0 {
		return ""
	}
	return ctx.PriorTurns[len(ctx.PriorTurns)-1].EndHEAD
}

// GitHEADAt returns current HEAD for project.
func GitHEADAt(cwd string) string {
	if cwd == "" {
		return ""
	}
	out, err := exec.Command("git", "-C", cwd, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
