package shellpreview

import (
	"strings"
	"testing"
)

func TestOneLinerSkipsPreamble(t *testing.T) {
	script := "set -euo pipefail\nBASE=\"$HOME/foo\"\ngo test ./...\necho done\n"
	got := OneLiner(script)
	if !strings.Contains(got, "go test") {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "set -euo") {
		t.Fatalf("should skip preamble: %q", got)
	}
}

func TestOneLinerJoinsContinuations(t *testing.T) {
	script := "set -euo pipefail\ninject() {\n  echo hi\n}\ncd /tmp && \\\ngo test ./internal/...\n"
	got := OneLiner(script)
	if strings.Contains(got, "inject()") {
		t.Fatalf("should skip function: %q", got)
	}
	if !strings.Contains(got, "go test") && !strings.Contains(got, "cd /tmp") {
		t.Fatalf("expected useful command: %q", got)
	}
}
