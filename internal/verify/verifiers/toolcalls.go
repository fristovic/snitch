package verifiers

import (
	"encoding/json"
	"strings"

	"github.com/fristovic/snitch/internal/transcript"
)

var readOnlyTools = map[string]bool{
	"Read": true, "Glob": true, "Grep": true, "SemanticSearch": true,
}

// ShellCommand returns the command string from a shell tool call.
func ShellCommand(tc transcript.ToolCall) string {
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

// PathFromInput extracts a file path from a tool call's input fields.
func PathFromInput(tc transcript.ToolCall, tool string) string {
	if p := transcript.PathFromToolInput(tc); p != "" {
		return p
	}
	_ = tool
	return ""
}

// HasMutating reports whether any tool call mutates project state.
func HasMutating(calls []transcript.ToolCall) bool {
	for _, tc := range calls {
		if !readOnlyTools[tc.Name] {
			return true
		}
	}
	return false
}

// HasGitCommitShell reports git commit commands in shell tool calls.
func HasGitCommitShell(calls []transcript.ToolCall) bool {
	for _, tc := range calls {
		if tc.Name != "Shell" {
			continue
		}
		cmd := strings.ToLower(ShellCommand(tc))
		if strings.Contains(cmd, "git commit") {
			return true
		}
	}
	return false
}

// HasGitPushShell reports git push commands in shell tool calls.
func HasGitPushShell(calls []transcript.ToolCall) bool {
	for _, tc := range calls {
		if tc.Name != "Shell" {
			continue
		}
		cmd := strings.ToLower(ShellCommand(tc))
		if strings.Contains(cmd, "git push") {
			return true
		}
	}
	return false
}

// HasTestShell reports test commands in shell tool calls.
func HasTestShell(calls []transcript.ToolCall) bool {
	for _, tc := range calls {
		if tc.Name != "Shell" {
			continue
		}
		if isTestCommand(ShellCommand(tc)) {
			return true
		}
	}
	return false
}

// HasShellCall reports any shell tool call.
func HasShellCall(calls []transcript.ToolCall) bool {
	for _, tc := range calls {
		if tc.Name == "Shell" {
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
