package verifiers

import "testing"

func TestLooksLikePath(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"foo.go", true},
		{"foo.go.", true},
		{"to", false},
		{"from", false},
		{"list", false},
		{"README", true},
		{"README.md", true},
		{"internal/foo.go", true},
		{"created a", false},
		{"commands", false},
		// Dotted code symbols are not file paths.
		{"Registry.ShellResolver", false},
		{"config.PlatformsConfig", false},
	}
	for _, tt := range tests {
		if got := LooksLikePath(tt.in); got != tt.want {
			t.Errorf("LooksLikePath(%q) = %v want %v", tt.in, got, tt.want)
		}
	}
}

func TestNormalizePathToken(t *testing.T) {
	if got := NormalizePathToken("foo.go."); got != "foo.go" {
		t.Fatalf("got %q", got)
	}
}

func TestPathsMatchTrailingPeriod(t *testing.T) {
	if !pathsMatch("foo.go", "foo.go.", "foo.go", "") {
		t.Fatal("expected match")
	}
}
