package transcript

import "testing"

func TestNewToolCall(t *testing.T) {
	tc := NewToolCall("Bash", claudeToolNames)
	if tc.RawName != "Bash" || tc.Name != ToolShell {
		t.Fatalf("got RawName=%q Name=%q", tc.RawName, tc.Name)
	}
	tc2 := NewToolCall("Shell", nil)
	if tc2.RawName != "Shell" || tc2.Name != "Shell" {
		t.Fatalf("cursor-style canonical raw: %+v", tc2)
	}
}
