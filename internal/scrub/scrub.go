package scrub

import (
	"strings"
	"unicode"
)

const redacted = "[REDACTED]"

// Scrub redacts secrets from a string using a single-pass scanner (no regex).
func Scrub(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		// Bearer token
		if _, ok := strings.CutPrefix(s[i:], "Bearer "); ok {
			b.WriteString("Bearer ")
			b.WriteString(redacted)
			i += len("Bearer ")
			j := i
			for j < len(s) && s[j] != ' ' && s[j] != '\n' && s[j] != '\r' {
				j++
			}
			i = j
			continue
		}
		// x-api-key header
		if _, ok := strings.CutPrefix(strings.ToLower(s[i:]), "x-api-key:"); ok {
			b.WriteString(s[i : i+len("x-api-key:")])
			b.WriteString(" ")
			b.WriteString(redacted)
			i += len("x-api-key:")
			for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
				i++
			}
			j := i
			for j < len(s) && s[j] != '\n' && s[j] != '\r' {
				j++
			}
			i = j
			continue
		}
		// --api-key and --token flags
		for _, flag := range []string{"--api-key", "--token"} {
			if strings.HasPrefix(s[i:], flag) {
				b.WriteString(flag)
				i += len(flag)
				if i < len(s) && s[i] == '=' {
					b.WriteByte('=')
					i++
				} else if i < len(s) && s[i] == ' ' {
					b.WriteByte(' ')
					i++
				}
				b.WriteString(redacted)
				for i < len(s) && s[i] != ' ' && s[i] != '\n' && s[i] != '\r' {
					i++
				}
				goto next
			}
		}
		// KEY=VALUE env style
		if eq := strings.IndexByte(s[i:], '='); eq > 0 && eq < 64 {
			key := s[i : i+eq]
			if looksLikeEnvKey(key) {
				valStart := i + eq + 1
				valEnd := valStart
				for valEnd < len(s) && s[valEnd] != ' ' && s[valEnd] != '\n' && s[valEnd] != '\r' {
					valEnd++
				}
				val := s[valStart:valEnd]
				if looksLikeSecret(val) {
					b.WriteString(key)
					b.WriteByte('=')
					b.WriteString(redacted)
					i = valEnd
					goto next
				}
			}
		}
		b.WriteByte(s[i])
		i++
	next:
	}
	return b.String()
}

func looksLikeEnvKey(key string) bool {
	if len(key) == 0 {
		return false
	}
	for _, r := range key {
		if !unicode.IsUpper(r) && r != '_' {
			return false
		}
	}
	return strings.Contains(key, "KEY") || strings.Contains(key, "TOKEN") || strings.Contains(key, "SECRET")
}

func looksLikeSecret(val string) bool {
	if len(val) < 20 {
		return false
	}
	prefixes := []string{"sk-", "pk-", "ghp_", "gho_", "ghu_", "ghs_", "ghr_", "xoxb-", "xoxp-"}
	for _, p := range prefixes {
		if strings.HasPrefix(val, p) {
			return true
		}
	}
	// High entropy heuristic: mostly alphanumeric
	alnum := 0
	for _, r := range val {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			alnum++
		}
	}
	return float64(alnum)/float64(len(val)) > 0.85
}
