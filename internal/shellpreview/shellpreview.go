// Package shellpreview extracts a short, human-useful command line from
// multi-line shell scripts for UI display (menu, log, notifications).
package shellpreview

import "strings"

// OneLiner returns the first executable-looking line of a multi-line shell
// script, skipping preambles, function bodies, and simple assignments.
func OneLiner(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = joinContinuations(s)
	var fallback string
	braceDepth := 0
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "\\" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if line == "set -euo pipefail" || line == "set -e" || line == "set -eu" || line == "set -o pipefail" {
			continue
		}
		if isFunctionDef(line) {
			braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
			if braceDepth < 0 {
				braceDepth = 0
			}
			continue
		}
		if braceDepth > 0 {
			braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
			if braceDepth < 0 {
				braceDepth = 0
			}
			continue
		}
		if strings.HasPrefix(line, "local ") {
			continue
		}
		if fallback == "" {
			fallback = line
		}
		if isAssignment(line) {
			continue
		}
		return line
	}
	if fallback != "" {
		return fallback
	}
	return strings.TrimSpace(s)
}

func joinContinuations(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	var buf string
	for _, line := range lines {
		trimRight := strings.TrimRight(line, " \t")
		if strings.HasSuffix(trimRight, "\\") {
			piece := strings.TrimSpace(strings.TrimSuffix(trimRight, "\\"))
			if buf == "" {
				buf = piece
			} else {
				buf += " " + piece
			}
			continue
		}
		piece := strings.TrimSpace(line)
		if buf != "" {
			if piece != "" {
				buf += " " + piece
			}
			out = append(out, buf)
			buf = ""
			continue
		}
		out = append(out, line)
	}
	if buf != "" {
		out = append(out, buf)
	}
	return strings.Join(out, "\n")
}

func isFunctionDef(line string) bool {
	if !strings.Contains(line, "(") || !strings.Contains(line, ")") {
		return false
	}
	idx := strings.IndexByte(line, '(')
	if idx <= 0 {
		return false
	}
	name := strings.TrimSpace(line[:idx])
	if name == "" {
		return false
	}
	for _, r := range name {
		if r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	rest := strings.TrimSpace(line[idx:])
	return strings.HasPrefix(rest, "()") || strings.HasPrefix(rest, "( )")
}

func isAssignment(line string) bool {
	eq := strings.IndexByte(line, '=')
	if eq <= 0 {
		return false
	}
	name := line[:eq]
	for _, r := range name {
		if r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}
