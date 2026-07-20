package transcript

import (
	"os"
	"path/filepath"
	"strings"
)

// PiPathResolver resolves project/session metadata from Pi session paths:
//
//	~/.pi/agent/sessions/--<encoded-cwd>--/<timestamp>_<uuid>.jsonl
//
// Pi encodes the working directory as a `--`-joined path segment in the
// session directory name (e.g. --Users-alice-projects-app--). The session id
// is the filename stem.
type PiPathResolver struct{}

func (PiPathResolver) ProjectCwd(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, p := range parts {
		// Directory segment wrapped in --...-- encodes the cwd.
		if strings.HasPrefix(p, "--") && strings.HasSuffix(p, "--") && len(p) > 4 {
			inner := strings.TrimSuffix(strings.TrimPrefix(p, "--"), "--")
			if inner == "" {
				return ""
			}
			return decodePiEncodedCwd(inner)
		}
	}
	return ""
}

func decodePiEncodedCwd(inner string) string {
	if decoded := decodePiEncodedCwdGreedy(inner); decoded != "" {
		return decoded
	}
	return "/" + strings.Join(strings.Split(inner, "-"), "/")
}

// decodePiEncodedCwdGreedy reconstructs a cwd from Pi's dash-encoded path by
// preferring directory boundaries that exist on disk, so embedded dashes in
// segment names (e.g. my-project) are preserved.
func decodePiEncodedCwdGreedy(inner string) string {
	segments := strings.Split(inner, "-")
	if len(segments) == 0 {
		return ""
	}
	path := "/" + segments[0]
	i := 1
	for i < len(segments) {
		matched := false
		for j := len(segments); j > i; j-- {
			name := strings.Join(segments[i:j], "-")
			candidate := filepath.Join(path, name)
			if _, err := os.Stat(candidate); err == nil {
				path = candidate
				i = j
				matched = true
				break
			}
		}
		if !matched {
			return ""
		}
	}
	return filepath.ToSlash(path)
}

func (PiPathResolver) ProjectDir(path string) string { return "" }

func (PiPathResolver) SessionID(path string) string { return sessionIDFromFilename(path, "") }
