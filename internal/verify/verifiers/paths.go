package verifiers

import (
	"path/filepath"
	"strings"
)

var pathStopwords = map[string]bool{
	"a": true, "an": true, "the": true, "to": true, "from": true, "list": true,
	"commands": true, "need": true, "legacy": true, "unused": true, "no-op": true,
	"documentation": true, "feature": true,
}

var knownBasenames = map[string]bool{
	"readme": true, "makefile": true, "dockerfile": true, "changelog": true,
	"license": true, "gemfile": true,
}

// NormalizePathToken trims quotes and trailing sentence punctuation from a captured path.
func NormalizePathToken(s string) string {
	s = strings.Trim(strings.TrimSpace(s), `"'`+"`")
	s = strings.TrimPrefix(s, "./")
	s = strings.TrimRight(s, ".,;:!?)]}")
	return s
}

// LooksLikePath reports whether a captured token plausibly names a file or directory path.
func LooksLikePath(target string) bool {
	target = NormalizePathToken(target)
	if target == "" {
		return false
	}
	lower := strings.ToLower(target)
	if pathStopwords[lower] {
		return false
	}
	if strings.Contains(target, "/") || strings.Contains(target, "\\") {
		return true
	}
	if strings.Contains(target, ".") {
		base := filepath.Base(target)
		if idx := strings.LastIndex(base, "."); idx > 0 && idx < len(base)-1 {
			return true
		}
	}
	if knownBasenames[lower] {
		return true
	}
	if strings.HasSuffix(lower, ".md") && len(target) > 3 {
		return true
	}
	return false
}
