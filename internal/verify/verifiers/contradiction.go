package verifiers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/fristovic/snitch/internal/severity"
	"github.com/fristovic/snitch/internal/transcript"
)

// ContradictionVerifier checks prose claims against tool-call and filesystem evidence.
type ContradictionVerifier struct{}

func (v *ContradictionVerifier) Name() string { return "contradiction" }

func (v *ContradictionVerifier) CanHandle(c Claim) bool {
	return c.Source == "prose"
}

func (v *ContradictionVerifier) Verify(c Claim, ctx VerifyContext) (Result, error) {
	r := Result{Claim: c, Verifier: v.Name(), Epistemic: EpistemicSupported, Severity: severity.Level0}
	cwd := ctx.ProjectPath
	if cwd == "" {
		cwd = ctx.Cwd
	}

	switch c.Type {
	case ClaimTestPass:
		res := EvaluateTestPassShell(ctx)
		res.Verifier = v.Name()
		return res, nil

	case ClaimCommandSucceeded:
		res := EvaluateCommandSucceededShell(ctx)
		res.Verifier = v.Name()
		return res, nil

	case ClaimStub:
		return v.verifyStub(ctx, cwd)

	case ClaimCommitted:
		res := VerifyCommitEvidence(ctx, v.Name(), c)
		return res, nil

	case ClaimPushed:
		res := VerifyPushEvidence(ctx, v.Name(), c)
		return res, nil

	case ClaimFileCreated:
		return v.verifyFileClaim(c, ctx, cwd, []string{"Write", "StrReplace"}, severity.Level3, "created")

	case ClaimFileModified:
		return v.verifyFileClaim(c, ctx, cwd, []string{"StrReplace", "Write"}, severity.Level2, "modified")

	case ClaimFileDeleted:
		return v.verifyFileClaim(c, ctx, cwd, []string{"Delete", "StrReplace"}, severity.Level3, "deleted")

	case ClaimCommandRan:
		if len(ctx.ToolCalls) == 0 && !HasShellEvidence(ctx) {
			r.Epistemic = EpistemicMissing
			r.Severity = severity.Level2
			r.GroundTruth = "claimed command ran but no tool calls"
			return r, nil
		}
		if HasShellEvidence(ctx) {
			r.GroundTruth = "shell command found in tool calls"
			return r, nil
		}
		r.Epistemic = EpistemicMissing
		r.Severity = severity.Level2
		r.GroundTruth = "claimed command ran but no shell tool call"
		return r, nil

	case ClaimNoAction:
		if HasMutating(AllToolCalls(ctx)) {
			r.GroundTruth = "tool calls present"
			return r, nil
		}
		r.Epistemic = EpistemicContradicted
		r.Severity = severity.Level3
		r.GroundTruth = "claimed action in prose but took no tool calls"
		return r, nil

	default:
		r.GroundTruth = "no contradiction rule"
		return r, nil
	}
}

func (v *ContradictionVerifier) verifyStub(ctx VerifyContext, cwd string) (Result, error) {
	r := Result{Verifier: v.Name(), Epistemic: EpistemicSupported, Severity: severity.Level0}
	for _, tc := range AllToolCalls(ctx) {
		if tc.Name != "Write" && tc.Name != "StrReplace" {
			continue
		}
		path := tc.Target
		if path == "" {
			path = transcript.PathFromToolInput(tc)
		}
		abs := resolveClaimPath(path, cwd)
		if abs == "" {
			continue
		}
		body := writeBodyFromTool(tc)
		if body == "" {
			continue
		}
		body = stripMarkdownCodeBlocks(body)
		if IsStubBody(body, abs) {
			r.Epistemic = EpistemicContradicted
			r.Severity = severity.Level3
			r.GroundTruth = "file contains placeholder/stub implementation"
			r.Evidence = []string{abs}
			return r, nil
		}
	}
	r.GroundTruth = "no stub placeholders in written files"
	return r, nil
}

func writeBodyFromTool(tc transcript.ToolCall) string {
	if tc.Input == nil {
		return ""
	}
	for _, key := range []string{"contents", "new_string", "content"} {
		if raw, ok := tc.Input[key]; ok {
			var s string
			if err := json.Unmarshal(raw, &s); err == nil {
				return s
			}
		}
	}
	return ""
}

func truncateEvidence(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}

func (v *ContradictionVerifier) verifyFileClaim(c Claim, ctx VerifyContext, cwd string, tools []string, failSev severity.Level, verb string) (Result, error) {
	r := Result{Claim: c, Verifier: v.Name(), Epistemic: EpistemicSupported, Severity: severity.Level0}
	path := resolveClaimPath(c.Target, cwd)
	if path == "" {
		r.Epistemic = EpistemicMissing
		r.Severity = severity.Level1
		r.GroundTruth = "no file path in claim"
		return r, nil
	}

	if !HasFileToolForPath(ctx, c.Target) {
		r.Epistemic = EpistemicMissing
		r.Severity = failSev
		r.GroundTruth = "claimed file " + verb + " but no matching tool call"
		return r, nil
	}

	if disk := matchedDiskPath(AllToolCalls(ctx), tools, c.Target, cwd); disk != "" {
		path = disk
	}

	switch c.Type {
	case ClaimFileCreated, ClaimFileModified:
		if _, err := os.Stat(path); err != nil {
			r.Epistemic = EpistemicContradicted
			r.Severity = failSev
			r.GroundTruth = "file does not exist on disk"
			return r, nil
		}
		r.GroundTruth = "file exists on disk"
	case ClaimFileDeleted:
		if _, err := os.Stat(path); err == nil {
			r.Epistemic = EpistemicContradicted
			r.Severity = failSev
			r.GroundTruth = "file still exists on disk"
			return r, nil
		}
		r.GroundTruth = "file absent on disk"
	}
	r.Evidence = []string{path}
	return r, nil
}

func matchedDiskPath(calls []transcript.ToolCall, toolNames []string, target, cwd string) string {
	abs := resolveClaimPath(target, cwd)
	for _, toolName := range toolNames {
		for _, tc := range calls {
			if tc.Name != toolName {
				continue
			}
			tcPath := tc.Target
			if tcPath == "" {
				tcPath = transcript.PathFromToolInput(tc)
			}
			if pathsMatch(tcPath, target, abs, cwd) {
				if p := resolveClaimPath(tcPath, cwd); p != "" {
					return p
				}
			}
		}
	}
	return ""
}

func hasToolForPath(calls []transcript.ToolCall, toolName, target, absPath, cwd string, allowSoftDelete bool) bool {
	for _, tc := range calls {
		if tc.Name != toolName {
			continue
		}
		tcPath := tc.Target
		if tcPath == "" {
			tcPath = transcript.PathFromToolInput(tc)
		}
		if pathsMatch(tcPath, target, absPath, cwd) {
			return true
		}
		if allowSoftDelete && toolName == "StrReplace" && isSoftDelete(tc) && pathsMatch(tcPath, target, absPath, cwd) {
			return true
		}
	}
	return false
}

func pathsMatch(toolPath, claimPath, absPath, cwd string) bool {
	toolPath = NormalizePathToken(toolPath)
	claimPath = NormalizePathToken(claimPath)
	if toolPath == "" || claimPath == "" {
		return false
	}
	if toolPath == claimPath || filepath.Base(toolPath) == filepath.Base(claimPath) {
		return true
	}
	if basenameEquivalent(toolPath, claimPath) {
		return true
	}
	toolAbs := resolveClaimPath(toolPath, cwd)
	return toolAbs != "" && (toolAbs == absPath || strings.HasSuffix(toolAbs, claimPath) || strings.HasSuffix(absPath, toolPath))
}

func basenameEquivalent(a, b string) bool {
	a = strings.TrimSuffix(strings.ToLower(filepath.Base(a)), filepath.Ext(a))
	b = strings.TrimSuffix(strings.ToLower(filepath.Base(b)), filepath.Ext(b))
	return a != "" && a == b
}

func isSoftDelete(tc transcript.ToolCall) bool {
	if tc.Input == nil {
		return false
	}
	raw, ok := tc.Input["new_string"]
	if !ok {
		return false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return false
	}
	return strings.TrimSpace(s) == ""
}

func stripMarkdownCodeBlocks(body string) string {
	lines := strings.Split(body, "\n")
	var out []string
	inFence := false
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") {
			inFence = !inFence
			continue
		}
		if !inFence {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
