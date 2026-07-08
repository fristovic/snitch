package transcript

import (
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
			return "/" + strings.Join(strings.Split(inner, "-"), "/")
		}
	}
	return ""
}

func (PiPathResolver) ProjectDir(path string) string { return "" }

func (PiPathResolver) SessionID(path string) string { return sessionIDFromFilename(path, "") }
