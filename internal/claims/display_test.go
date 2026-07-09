package claims

import (
	"strings"
	"testing"
)

func TestClaimTypeLabel(t *testing.T) {
	if got := ClaimTypeLabel(TypeNoAction); got != "No action taken" {
		t.Fatalf("no_action: got %q", got)
	}
	if got := ClaimTypeLabel(TypeToolWrite); got != "File write" {
		t.Fatalf("tool_write: got %q", got)
	}
	if got := ClaimTypeLabel("Write"); got != "File write" {
		t.Fatalf("legacy Write: got %q", got)
	}
}

func TestFlaggedTextPrefersSentence(t *testing.T) {
	d := DisplayFields{
		ClaimType:     TypeNoAction,
		Claimed:       "updated",
		ClaimSentence: "I updated the README.",
	}
	if got := FlaggedText(d); got != "I updated the README." {
		t.Fatalf("got %q", got)
	}
}

func TestShortSummary(t *testing.T) {
	d := DisplayFields{
		ClaimType:     TypeTestPass,
		ClaimSentence: "All tests pass on main.",
	}
	got := ShortSummary(d, 40)
	if !strings.Contains(got, "Tests passed") {
		t.Fatalf("missing label: %q", got)
	}
	if !strings.Contains(got, "All tests pass") {
		t.Fatalf("missing flagged text: %q", got)
	}
}

func TestNotificationBody(t *testing.T) {
	d := DisplayFields{
		ClaimType:     TypeNoAction,
		ClaimSentence: "I updated README and committed.",
		Actual:        "claimed action in prose but took no tool calls",
	}
	got := NotificationBody(d, 120)
	if !strings.Contains(got, "I updated README") {
		t.Fatalf("missing flagged: %q", got)
	}
	if !strings.Contains(got, "no tool calls") {
		t.Fatalf("missing actual: %q", got)
	}
}

func TestRichDetail(t *testing.T) {
	d := DisplayFields{
		ClaimType:     TypeTestPass,
		Source:        "prose",
		ClaimSentence: "All tests pass.",
		ClaimContext:  "Great news. All tests pass. Shipping now.",
		Actual:        "claimed tests pass but ran no tests",
		Verifier:      "contradiction",
		Severity:      3,
		Evidence:      []string{"no test tool calls"},
	}
	got := RichDetail(d)
	for _, want := range []string{"Flagged:", "Checked:", "Context:", "Evidence:", "Verifier:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}

func TestFlaggedTextCompactsLongShell(t *testing.T) {
	script := "Shell set -euo pipefail\nBASE=\"$HOME/.cursor/projects/foo\"\nTS=$(date +%s)\ngo test ./...\necho done\n"
	d := DisplayFields{
		ClaimType:     TypeToolShell,
		Source:        "tool",
		Claimed:       script,
		ClaimSentence: script,
		Target:        "set -euo pipefail\nBASE=\"$HOME/.cursor/projects/foo\"\ngo test ./...",
		Actual:        "test output indicates failure",
	}
	got := FlaggedText(d)
	if strings.Contains(got, "\n") {
		t.Fatalf("expected one line, got %q", got)
	}
	if !strings.Contains(got, "go test") {
		t.Fatalf("expected go test line, got %q", got)
	}
	if strings.Contains(got, "set -euo pipefail") {
		t.Fatalf("should skip preamble, got %q", got)
	}
	if len([]rune(got)) > maxToolFlaggedRunes {
		t.Fatalf("too long (%d): %q", len([]rune(got)), got)
	}

	d.Target = ""
	got = FlaggedText(d)
	if !strings.Contains(got, "go test") {
		t.Fatalf("expected go test line, got %q", got)
	}

	// Continuations + function defs should not win.
	d.Claimed = "Shell set -euo pipefail\ninject() {\n  echo hi\n}\ncd /tmp && \\\ngo test ./internal/...\n"
	d.ClaimSentence = d.Claimed
	d.Target = ""
	got = FlaggedText(d)
	if strings.Contains(got, "inject()") {
		t.Fatalf("should skip function def, got %q", got)
	}
	if !strings.Contains(got, "go test") && !strings.Contains(got, "cd /tmp") {
		t.Fatalf("expected joined/useful command, got %q", got)
	}
}

func TestRichDetailToolDoesNotDumpScript(t *testing.T) {
	long := strings.Repeat("echo line\n", 200)
	d := DisplayFields{
		ClaimType:     TypeToolShell,
		Source:        "tool",
		Claimed:       "Shell " + long,
		ClaimSentence: "Shell " + long,
		Target:        long,
		Actual:        "test output indicates failure",
		Verifier:      "shell",
		Severity:      2,
	}
	got := RichDetail(d)
	if strings.Count(got, "\n") > 6 {
		t.Fatalf("too many lines:\n%s", got)
	}
	if strings.Contains(got, "Context:") {
		t.Fatalf("tool claims should omit context:\n%s", got)
	}
	if !strings.Contains(got, "Flagged:") || !strings.Contains(got, "Checked:") {
		t.Fatalf("missing fields:\n%s", got)
	}
}
