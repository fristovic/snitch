package transcript

import (
	"path/filepath"
	"strings"
)

// ProjectCwdFromTranscriptPath derives project cwd from a Cursor transcript path.
func ProjectCwdFromTranscriptPath(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i, p := range parts {
		if p == "projects" && i+1 < len(parts) {
			return DecodeProjectSlug(parts[i+1])
		}
	}
	return ""
}

// DecodeProjectSlug maps Cursor project slugs like Users-a-b-c to /Users/a/b/c.
func DecodeProjectSlug(slug string) string {
	if !strings.HasPrefix(slug, "Users-") {
		return ""
	}
	segments := strings.Split(slug, "-")
	if len(segments) < 2 {
		return ""
	}
	return "/" + strings.Join(segments, "/")
}

// SessionIDFromTranscriptPath returns the session UUID from a transcript file path.
// Layout: .../agent-transcripts/<sessionUUID>/<sessionUUID>.jsonl
func SessionIDFromTranscriptPath(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i, p := range parts {
		if p == "agent-transcripts" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// IsSubagentTranscript reports whether path is under a subagents/ directory.
func IsSubagentTranscript(path string) bool {
	return strings.Contains(filepath.ToSlash(path), "/subagents/")
}
