package claims

import "testing"

func TestToolNameToClaimType(t *testing.T) {
	tests := []struct {
		name string
		want string
		ok   bool
	}{
		{"Write", TypeToolWrite, true},
		{"StrReplace", TypeToolStrReplace, true},
		{"Shell", TypeToolShell, true},
		{"Unknown", "", false},
	}
	for _, tt := range tests {
		got, ok := ToolNameToClaimType(tt.name)
		if ok != tt.ok || got != tt.want {
			t.Fatalf("%s: got (%q, %v), want (%q, %v)", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}

func TestNormalizeClaimType(t *testing.T) {
	if got := NormalizeClaimType("Write"); got != TypeToolWrite {
		t.Fatalf("Write: got %q", got)
	}
	if got := NormalizeClaimType(TypeToolWrite); got != TypeToolWrite {
		t.Fatalf("tool_write: got %q", got)
	}
	if got := NormalizeClaimType("no_action"); got != TypeNoAction {
		t.Fatalf("no_action: got %q", got)
	}
}

func TestAllFilterTypesOrder(t *testing.T) {
	types := AllFilterTypes()
	if len(types) < 10 {
		t.Fatalf("expected many filter types, got %d", len(types))
	}
	if types[0] != TypeTestPass {
		t.Fatalf("first should be test_pass, got %q", types[0])
	}
	foundTool := false
	for _, ty := range types {
		if ty == TypeToolWrite {
			foundTool = true
			break
		}
	}
	if !foundTool {
		t.Fatal("missing tool_write in filter types")
	}
}
