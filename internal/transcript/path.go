package transcript

import (
	"os"
	"path/filepath"
	"strings"
)

// slugProjectCwd finds the /projects/<slug>/ segment in path and decodes the
// slug to an absolute cwd. Shared by the Cursor and Claude resolvers, whose
// layouts differ only in the app directory (.cursor vs .claude).
func slugProjectCwd(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i, p := range parts {
		if p == "projects" && i+1 < len(parts) {
			return DecodeProjectSlug(parts[i+1])
		}
	}
	return ""
}

// slugProjectDir returns ~/<appDir>/projects/<slug> for a transcript path.
func slugProjectDir(path, appDir string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i, p := range parts {
		if p == "projects" && i+1 < len(parts) {
			home, err := os.UserHomeDir()
			if err != nil {
				return ""
			}
			return filepath.Join(home, appDir, "projects", parts[i+1])
		}
	}
	return ""
}

// sessionIDFromFilename returns the filename stem, optionally stripping a
// prefix (Codex "rollout-").
func sessionIDFromFilename(path, stripPrefix string) string {
	base := filepath.Base(path)
	base = strings.TrimPrefix(base, stripPrefix)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// DecodeProjectSlug maps project path slugs to absolute paths.
// Cursor uses Users-a-b-c; Claude Code uses -Users-a-b-c (leading dash).
func DecodeProjectSlug(slug string) string {
	slug = strings.TrimPrefix(slug, "-")
	if !strings.HasPrefix(slug, "Users-") {
		return ""
	}
	segments := strings.Split(slug, "-")
	if len(segments) < 2 {
		return ""
	}
	return "/" + strings.Join(segments, "/")
}

// CursorPathResolver resolves project/session metadata from Cursor transcript
// paths, which follow ~/.cursor/projects/<slug>/agent-transcripts/<uuid>/...
type CursorPathResolver struct{}

func (CursorPathResolver) ProjectCwd(path string) string { return slugProjectCwd(path) }

func (CursorPathResolver) ProjectDir(path string) string { return slugProjectDir(path, ".cursor") }

func (CursorPathResolver) SessionID(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i, p := range parts {
		if p == "agent-transcripts" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
