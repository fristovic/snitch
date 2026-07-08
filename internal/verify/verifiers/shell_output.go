package verifiers

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/fristovic/snitch/internal/transcript"
)

// ShellOutputForCommand returns captured output for a shell tool call.
// Resolution order: inline tool_result on the call, then harness-specific
// shell artifacts via ctx.ShellOutputResolver (Cursor terminal files for
// cursor; no-op for harnesses that embed output inline). A relaxed time window
// retry covers terminals that finish slightly after the turn.
func ShellOutputForCommand(tc transcript.ToolCall, ctx VerifyContext) (output string, exitCode int, found bool) {
	// Step 1: inline tool_result (harness-agnostic).
	if tc.Result != "" || tc.IsError {
		code := 0
		if tc.IsError {
			code = 1
		}
		return tc.Result, code, true
	}
	// Step 2: extract the command string.
	cmd := ShellCommand(tc)
	if cmd == "" {
		return "", 0, false
	}
	resolver := ctx.ShellOutputResolver
	if resolver == nil {
		return "", 0, false
	}
	// Step 3: harness-specific resolution, tight window.
	req := transcript.ShellOutputRequest{
		TranscriptPath: ctx.TranscriptPath,
		ProjectPath:    ctx.ProjectPath,
		Cwd:            ctx.Cwd,
		Command:        cmd,
		StartedAt:      ctx.StartedAt,
		FinishedAt:     ctx.FinishedAt,
	}
	if out, code, ok := resolver.Resolve(tc, req); ok {
		return out, code, true
	}
	// Step 4: relaxed window (terminal may finish slightly after turn).
	req.StartedAt = ctx.StartedAt.Add(-5 * time.Minute)
	req.FinishedAt = ctx.FinishedAt.Add(10 * time.Minute)
	return resolver.Resolve(tc, req)
}

// ParseTestOutput inspects command output for pass/fail signals.
func ParseTestOutput(output string) (passed bool, found bool) {
	for _, parser := range testParsers {
		if passed, found := parser(output); found {
			return passed, true
		}
	}
	return false, false
}

var testParsers = []func(string) (passed bool, found bool){
	parseGoTestOutput,
	parsePytestOutput,
	parseVitestOutput,
	parseCargoTestOutput,
	parseNpmTestOutput,
	parseGenericTestOutput,
}

func parseGoTestOutput(output string) (bool, bool) {
	if strings.Contains(output, "--- FAIL:") || strings.Contains(output, "FAIL\t") {
		return false, true
	}
	if strings.Contains(output, "ok\t") || strings.Contains(output, "\tok\t") || strings.Contains(output, "ok  \t") {
		return true, true
	}
	return false, false
}

func parsePytestOutput(output string) (bool, bool) {
	lower := strings.ToLower(output)
	if strings.Contains(lower, "===== ") && strings.Contains(lower, " failed") {
		return false, true
	}
	if strings.Contains(lower, "failures") && !strings.Contains(lower, "0 failed") {
		return false, true
	}
	if strings.Contains(lower, "===== ") && strings.Contains(lower, " passed") {
		return true, true
	}
	return false, false
}

func parseVitestOutput(output string) (bool, bool) {
	lower := strings.ToLower(output)
	if strings.Contains(lower, " failed") && !strings.Contains(lower, "0 failed") {
		return false, true
	}
	if strings.Contains(lower, "test files") && strings.Contains(lower, " passed") {
		return true, true
	}
	return false, false
}

func parseCargoTestOutput(output string) (bool, bool) {
	lower := strings.ToLower(output)
	if strings.Contains(lower, "test result: failed") || strings.Contains(lower, "error: test failed") {
		return false, true
	}
	if strings.Contains(lower, "test result: ok") {
		return true, true
	}
	return false, false
}

func parseNpmTestOutput(output string) (bool, bool) {
	lower := strings.ToLower(output)
	if strings.Contains(lower, "tests:") && strings.Contains(lower, "failed") && !strings.Contains(lower, "0 failed") {
		return false, true
	}
	if strings.Contains(lower, "fail 1 test") {
		return false, true
	}
	if strings.Contains(lower, "tests:") && strings.Contains(lower, "passed") && !strings.Contains(lower, "failed") {
		return true, true
	}
	return false, false
}

func parseGenericTestOutput(output string) (bool, bool) {
	lower := strings.ToLower(output)
	if strings.Contains(lower, "build failed") ||
		strings.Contains(lower, "--- fail") ||
		strings.Contains(lower, "test failed") ||
		strings.Contains(lower, "failures") ||
		strings.Contains(lower, "fail ") ||
		strings.Contains(lower, "\tfail") ||
		strings.HasPrefix(lower, "fail") ||
		(strings.Contains(lower, "failed") && !strings.Contains(lower, "0 failed")) {
		return false, true
	}
	if strings.Contains(lower, "ok\t") ||
		strings.Contains(lower, "passed") ||
		strings.Contains(lower, "all tests passed") ||
		(strings.Contains(lower, "exit_code: 0") && strings.Contains(lower, "test")) {
		return true, true
	}
	return false, false
}

// IsStubBody reports whether file contents look like an unimplemented placeholder.
func IsStubBody(body, path string) bool {
	trim := strings.TrimSpace(body)
	if trim == "" {
		return true
	}
	ext := strings.ToLower(filepath.Ext(path))
	if trim == "pass" && ext != ".go" && ext != ".py" {
		return false
	}
	if trim == "pass" || trim == "..." {
		return true
	}
	if isDocPath(path) {
		if strings.Contains(strings.ToLower(trim), "fixme") || strings.Contains(strings.ToLower(trim), "// todo") {
			return false
		}
		return isStubInCodeOnly(trim)
	}
	lower := strings.ToLower(trim)
	stubs := []string{
		`panic("todo")`, `panic('todo')`, `panic("not implemented")`,
		"notimplementederror", `throw new error("not implemented")`,
		"// todo", "# todo", "todo:", "fixme", "unimplemented",
	}
	for _, s := range stubs {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return isEmptyImplementation(lower)
}

func isDocPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".txt", ".rst", ".adoc":
		return true
	default:
		base := strings.ToLower(filepath.Base(path))
		return base == "changelog" || strings.HasPrefix(base, "readme")
	}
}

func isStubInCodeOnly(body string) bool {
	lower := strings.ToLower(body)
	if strings.Contains(lower, "example") || strings.Contains(lower, "```") || strings.Contains(lower, "|") {
		return false
	}
	return strings.Contains(lower, `panic("todo")`) || strings.Contains(lower, `panic('todo')`)
}

func isEmptyImplementation(lower string) bool {
	if strings.Contains(lower, "return nil") && strings.Count(lower, "\n") < 4 {
		return true
	}
	if strings.TrimSpace(lower) == "..." {
		return true
	}
	return false
}
