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
		if HasCommitEvidence(ctx) {
			r.Accurate = true
			r.GroundTruth = "commit evidence found"
			return r, nil
		}
		r.Accurate = false
		r.Severity = severity.Level3
		r.GroundTruth = "claimed commit but no commit evidence"
		return r, nil

	case ClaimPushed:
		if HasPushEvidence(ctx) {
			r.Accurate = true
			r.GroundTruth = "git push shell call found"
			return r, nil
		}
		r.Accurate = false
		r.Severity = severity.Level2
		r.GroundTruth = "claimed push but no git push command"
		return r, nil

	case ClaimFileCreated:
		return v.verifyFileClaim(c, ctx, cwd, []string{"Write", "StrReplace"}, severity.Level3, "created")

	case ClaimFileModified:
		return v.verifyFileClaim(c, ctx, cwd, []string{"StrReplace", "Write"}, severity.Level2, "modified")

	case ClaimFileDeleted:
		return v.verifyFileClaim(c, ctx, cwd, []string{"Delete", "StrReplace"}, severity.Level3, "deleted")

	case ClaimCommandRan:
		if len(ctx.ToolCalls) == 0 && !HasShellEvidence(ctx) {
			r.Accurate = false
			r.Severity = severity.Level2
			r.GroundTruth = "claimed command ran but no tool calls"
			return r, nil
		}
		if HasShellEvidence(ctx) {
			r.Accurate = true
			r.GroundTruth = "shell command found in tool calls"
			return r, nil
		}
		r.Accurate = false
		r.Severity = severity.Level2
		r.GroundTruth = "claimed command ran but no shell tool call"
		return r, nil

	case ClaimNoAction:
		if hasMutatingToolCalls(ctx.ToolCalls) {
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
	calls := AllToolCalls(ctx)
	if !HasTestShellEvidence(ctx) {
		r.Accurate = false
		r.Severity = severity.Level3
		r.GroundTruth = "claimed tests pass but ran no tests"
		return r, nil
	}
	for _, tc := range calls {
		if tc.Name != "Shell" || !isTestCommand(shellCommand(tc)) {
			continue
		}
		out, code, found := ShellOutputForCommand(tc, ctx)
		if !found {
			if tc.Result == "" && !tc.IsError {
				continue
			}
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
	calls := AllToolCalls(ctx)
	if !HasShellEvidence(ctx) {
		r.Accurate = false
		r.Severity = severity.Level3
		r.GroundTruth = "claimed success but no shell command ran"
		return r, nil
	}
	for _, tc := range calls {
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
		if passed, ok := ParseTestOutput(out); ok && !passed {
			r.Accurate = false
			r.Severity = severity.Level3
			r.GroundTruth = "shell output indicates failure"
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
	for _, tc := range AllToolCalls(ctx) {
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
			continue
		}
		body = stripMarkdownCodeBlocks(body)
		if IsStubBody(body, abs) {
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

func (v *ContradictionVerifier) verifyFileClaim(c Claim, ctx VerifyContext, cwd string, tools []string, failSev severity.Level, verb string) (Result, error) {
	r := Result{Claim: c, Verifier: v.Name(), Severity: severity.Level0}
	path := resolveClaimPath(c.Target, cwd)
	if path == "" {
		r.Accurate = false
		r.Severity = severity.Level1
		r.GroundTruth = "no file path in claim"
		return r, nil
	}

	if !HasFileToolForPath(ctx, c.Target) {
		r.Accurate = false
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

func matchedDiskPath(calls []transcript.ToolCall, toolNames []string, target, cwd string) string {
	abs := resolveClaimPath(target, cwd)
	for _, toolName := range toolNames {
		for _, tc := range calls {
			if tc.Name != toolName {
				continue
			}
			tcPath := tc.Target
			if tcPath == "" {
				tcPath = pathFromInput(tc, toolName)
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

func hasGitActivityShell(calls []transcript.ToolCall) bool {
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

func hasAnyToolForPath(calls []transcript.ToolCall, toolNames []string, target, absPath, cwd string, allowSoftDelete bool) bool {
	for _, toolName := range toolNames {
		if hasToolForPath(calls, toolName, target, absPath, cwd, allowSoftDelete) {
			return true
		}
	}
	return false
}

func hasToolForPath(calls []transcript.ToolCall, toolName, target, absPath, cwd string, allowSoftDelete bool) bool {
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
		if allowSoftDelete && toolName == "StrReplace" && isSoftDelete(tc) && pathsMatch(tcPath, target, absPath, cwd) {
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
