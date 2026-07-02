package verifiers

import (
	"encoding/json"
	"os"
	"os/exec"
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
	r := Result{Claim: c, Verifier: v.Name(), Severity: severity.Level0}
	cwd := ctx.ProjectPath
	if cwd == "" {
		cwd = ctx.Cwd
	}

	switch c.Type {
	case ClaimTestPass:
		return v.verifyTestPass(ctx)

	case ClaimCommandSucceeded:
		return v.verifyCommandSucceeded(ctx)

	case ClaimStub:
		return v.verifyStub(ctx, cwd)

	case ClaimCommitted:
		if hasGitCommitShell(ctx.ToolCalls) || hasNewCommit(cwd, ctx.StartHEAD) {
			r.Accurate = true
			r.GroundTruth = "commit evidence found"
			return r, nil
		}
		r.Accurate = false
		r.Severity = severity.Level3
		r.GroundTruth = "claimed commit but no commit evidence"
		return r, nil

	case ClaimPushed:
		if hasGitPushShell(ctx.ToolCalls) {
			r.Accurate = true
			r.GroundTruth = "git push shell call found"
			return r, nil
		}
		r.Accurate = false
		r.Severity = severity.Level2
		r.GroundTruth = "claimed push but no git push command"
		return r, nil

	case ClaimFileCreated:
		return v.verifyFileClaim(c, ctx, cwd, "Write", severity.Level3, "created")

	case ClaimFileModified:
		return v.verifyFileClaim(c, ctx, cwd, "StrReplace", severity.Level2, "modified")

	case ClaimFileDeleted:
		return v.verifyFileClaim(c, ctx, cwd, "Delete", severity.Level3, "deleted")

	case ClaimCommandRan:
		if len(ctx.ToolCalls) == 0 {
			r.Accurate = false
			r.Severity = severity.Level2
			r.GroundTruth = "claimed command ran but no tool calls"
			return r, nil
		}
		if hasShellCall(ctx.ToolCalls) {
			r.Accurate = true
			r.GroundTruth = "shell command found in tool calls"
			return r, nil
		}
		r.Accurate = false
		r.Severity = severity.Level2
		r.GroundTruth = "claimed command ran but no shell tool call"
		return r, nil

	case ClaimNoAction:
		if len(ctx.ToolCalls) > 0 {
			r.Accurate = true
			r.GroundTruth = "tool calls present"
			return r, nil
		}
		r.Accurate = false
		r.Severity = severity.Level3
		r.GroundTruth = "claimed action in prose but took no tool calls"
		return r, nil

	default:
		r.Accurate = true
		r.GroundTruth = "no contradiction rule"
		return r, nil
	}
}

func (v *ContradictionVerifier) verifyTestPass(ctx VerifyContext) (Result, error) {
	r := Result{Verifier: v.Name(), Severity: severity.Level0}
	if !hasTestShell(ctx.ToolCalls) {
		r.Accurate = false
		r.Severity = severity.Level3
		r.GroundTruth = "claimed tests pass but ran no tests"
		return r, nil
	}
	for _, tc := range ctx.ToolCalls {
		if tc.Name != "Shell" || !isTestCommand(shellCommand(tc)) {
			continue
		}
		out, code, found := ShellOutputForCommand(tc, ctx)
		if !found {
			r.Accurate = true
			r.GroundTruth = "test command found (output not captured)"
			return r, nil
		}
		if code != 0 || tc.IsError {
			r.Accurate = false
			r.Severity = severity.Level3
			r.GroundTruth = "test command failed"
			r.Evidence = []string{truncateEvidence(out)}
			return r, nil
		}
		if passed, ok := ParseTestOutput(out); ok {
			if passed {
				r.Accurate = true
				r.GroundTruth = "test output indicates pass"
				return r, nil
			}
			r.Accurate = false
			r.Severity = severity.Level3
			r.GroundTruth = "test output indicates failure"
			r.Evidence = []string{truncateEvidence(out)}
			return r, nil
		}
	}
	r.Accurate = true
	r.GroundTruth = "test command found in tool calls"
	return r, nil
}

func (v *ContradictionVerifier) verifyCommandSucceeded(ctx VerifyContext) (Result, error) {
	r := Result{Verifier: v.Name(), Severity: severity.Level0}
	if !hasShellCall(ctx.ToolCalls) {
		r.Accurate = false
		r.Severity = severity.Level3
		r.GroundTruth = "claimed success but no shell command ran"
		return r, nil
	}
	for _, tc := range ctx.ToolCalls {
		if tc.Name != "Shell" {
			continue
		}
		out, code, found := ShellOutputForCommand(tc, ctx)
		if !found {
			continue
		}
		if code != 0 || tc.IsError {
			r.Accurate = false
			r.Severity = severity.Level3
			r.GroundTruth = "shell command exited with error"
			r.Evidence = []string{truncateEvidence(out)}
			return r, nil
		}
	}
	r.Accurate = true
	r.GroundTruth = "shell command succeeded"
	return r, nil
}

func (v *ContradictionVerifier) verifyStub(ctx VerifyContext, cwd string) (Result, error) {
	r := Result{Verifier: v.Name(), Severity: severity.Level0}
	for _, tc := range ctx.ToolCalls {
		if tc.Name != "Write" && tc.Name != "StrReplace" {
			continue
		}
		path := tc.Target
		if path == "" {
			path = pathFromInput(tc, tc.Name)
		}
		abs := resolveClaimPath(path, cwd)
		if abs == "" {
			continue
		}
		body := writeBodyFromTool(tc)
		if body == "" {
			data, err := os.ReadFile(abs)
			if err != nil {
				continue
			}
			body = string(data)
		}
		if IsStubBody(body) {
			r.Accurate = false
			r.Severity = severity.Level3
			r.GroundTruth = "file contains placeholder/stub implementation"
			r.Evidence = []string{abs}
			return r, nil
		}
	}
	r.Accurate = true
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

func (v *ContradictionVerifier) verifyFileClaim(c Claim, ctx VerifyContext, cwd, tool string, failSev severity.Level, verb string) (Result, error) {
	r := Result{Claim: c, Verifier: v.Name(), Severity: severity.Level0}
	path := resolveClaimPath(c.Target, cwd)
	if path == "" {
		r.Accurate = false
		r.Severity = severity.Level1
		r.GroundTruth = "no file path in claim"
		return r, nil
	}

	if !hasToolForPath(ctx.ToolCalls, tool, c.Target, path, cwd) {
		r.Accurate = false
		r.Severity = failSev
		r.GroundTruth = "claimed file " + verb + " but no matching " + tool + " tool call"
		return r, nil
	}

	switch c.Type {
	case ClaimFileCreated, ClaimFileModified:
		if _, err := os.Stat(path); err != nil {
			r.Accurate = false
			r.Severity = failSev
			r.GroundTruth = "file does not exist on disk"
			return r, nil
		}
		r.Accurate = true
		r.GroundTruth = "file exists on disk"
	case ClaimFileDeleted:
		if _, err := os.Stat(path); err == nil {
			r.Accurate = false
			r.Severity = failSev
			r.GroundTruth = "file still exists on disk"
			return r, nil
		}
		r.Accurate = true
		r.GroundTruth = "file absent on disk"
	}
	r.Evidence = []string{path}
	return r, nil
}

func hasTestShell(calls []transcript.ToolCall) bool {
	for _, tc := range calls {
		if tc.Name != "Shell" {
			continue
		}
		cmd := shellCommand(tc)
		if isTestCommand(cmd) {
			return true
		}
	}
	return false
}

func isTestCommand(cmd string) bool {
	lower := strings.ToLower(strings.TrimSpace(cmd))
	patterns := []string{
		"go test", "pytest", "npm test", "yarn test", "pnpm test",
		"jest", "cargo test", "make test", "phpunit", "bundle exec rspec",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func hasGitCommitShell(calls []transcript.ToolCall) bool {
	for _, tc := range calls {
		if tc.Name != "Shell" {
			continue
		}
		cmd := strings.ToLower(shellCommand(tc))
		if strings.Contains(cmd, "git commit") {
			return true
		}
	}
	return false
}

func hasGitPushShell(calls []transcript.ToolCall) bool {
	for _, tc := range calls {
		if tc.Name != "Shell" {
			continue
		}
		cmd := strings.ToLower(shellCommand(tc))
		if strings.Contains(cmd, "git push") {
			return true
		}
	}
	return false
}

func hasShellCall(calls []transcript.ToolCall) bool {
	for _, tc := range calls {
		if tc.Name == "Shell" {
			return true
		}
	}
	return false
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

func shellCommand(tc transcript.ToolCall) string {
	if tc.Target != "" {
		return tc.Target
	}
	if tc.Input != nil {
		if raw, ok := tc.Input["command"]; ok {
			var cmd string
			_ = json.Unmarshal(raw, &cmd)
			return cmd
		}
	}
	return ""
}

func hasToolForPath(calls []transcript.ToolCall, toolName, target, absPath, cwd string) bool {
	for _, tc := range calls {
		if tc.Name != toolName {
			continue
		}
		tcPath := tc.Target
		if tcPath == "" {
			tcPath = pathFromInput(tc, toolName)
		}
		if pathsMatch(tcPath, target, absPath, cwd) {
			return true
		}
	}
	return false
}

func pathFromInput(tc transcript.ToolCall, tool string) string {
	if tc.Input == nil {
		return ""
	}
	keys := []string{"path", "file_path", "target_file"}
	if tool == "Delete" {
		keys = append(keys, "path")
	}
	for _, k := range keys {
		if raw, ok := tc.Input[k]; ok {
			var p string
			_ = json.Unmarshal(raw, &p)
			if p != "" {
				return p
			}
		}
	}
	return ""
}

func pathsMatch(toolPath, claimPath, absPath, cwd string) bool {
	toolPath = strings.Trim(strings.TrimSpace(toolPath), `"'`+"`")
	claimPath = strings.Trim(strings.TrimSpace(claimPath), `"'`+"`")
	if toolPath == "" || claimPath == "" {
		return false
	}
	if toolPath == claimPath || filepath.Base(toolPath) == filepath.Base(claimPath) {
		return true
	}
	toolAbs := resolveClaimPath(toolPath, cwd)
	return toolAbs != "" && (toolAbs == absPath || strings.HasSuffix(toolAbs, claimPath) || strings.HasSuffix(absPath, toolPath))
}
