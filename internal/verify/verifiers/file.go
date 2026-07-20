package verifiers

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fristovic/snitch/internal/severity"
)

// FileVerifier checks file-related tool calls.
type FileVerifier struct{}

func (v *FileVerifier) Name() string { return "file" }

func (v *FileVerifier) CanHandle(c Claim) bool {
	if c.Source != "tool" {
		return false
	}
	switch c.Type {
	case ClaimToolWrite, ClaimToolStrReplace, ClaimToolDelete, ClaimToolRead, ClaimToolGlob:
		return true
	default:
		return false
	}
}

func (v *FileVerifier) Verify(c Claim, ctx VerifyContext) (Result, error) {
	r := Result{Claim: c, Verifier: v.Name(), Epistemic: EpistemicSupported, Severity: severity.Level0}
	path := resolveClaimPath(c.Target, ctx.Cwd)
	if path == "" {
		r.Epistemic = EpistemicMissing
		r.Severity = severity.Level1
		r.GroundTruth = "no path in tool call"
		return r, nil
	}

	switch c.Type {
	case ClaimToolDelete:
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			r.GroundTruth = "file deleted"
			return r, nil
		}
		r.Epistemic = EpistemicContradicted
		r.Severity = severity.Level2
		r.GroundTruth = "file still exists"
		return r, nil

	case ClaimToolRead:
		if _, err := os.Stat(path); err != nil {
			r.Epistemic = EpistemicContradicted
			r.Severity = severity.Level1
			r.GroundTruth = "file does not exist"
			return r, nil
		}
		r.GroundTruth = "file exists"
		r.Evidence = []string{path}
		return r, nil

	case ClaimToolGlob:
		pattern := c.Target
		if pattern == "" {
			if s, ok := c.Input["glob_pattern"].(string); ok {
				pattern = s
			}
		}
		if !filepath.IsAbs(pattern) && ctx.Cwd != "" {
			pattern = filepath.Join(ctx.Cwd, pattern)
		}
		matches, _ := filepath.Glob(pattern)
		r.GroundTruth = "glob matched " + strconv.Itoa(len(matches)) + " paths"
		return r, nil

	case ClaimToolStrReplace:
		info, err := os.Stat(path)
		if err != nil {
			r.Epistemic = EpistemicContradicted
			r.Severity = severity.Level3
			r.GroundTruth = "file does not exist"
			return r, nil
		}
		newStr, _ := c.Input["new_string"].(string)
		if newStr != "" {
			data, err := os.ReadFile(path)
			if err != nil {
				r.Epistemic = EpistemicMissing
				r.Severity = severity.Level2
				r.GroundTruth = "cannot read file"
				return r, nil
			}
			if !strings.Contains(string(data), newStr) {
				r.Epistemic = EpistemicContradicted
				r.Severity = severity.Level2
				r.GroundTruth = "new_string not found in file"
				return r, nil
			}
		}
		r.GroundTruth = "file exists (" + formatSize(info.Size()) + " bytes), content check passed"
		r.Evidence = []string{path}
		return r, nil

	case ClaimToolWrite:
		info, err := os.Stat(path)
		if err != nil {
			r.Epistemic = EpistemicContradicted
			r.Severity = severity.Level3
			r.GroundTruth = "file does not exist"
			return r, nil
		}
		if info.Size() < 10 {
			r.Epistemic = EpistemicContradicted
			r.Severity = severity.Level2
			r.GroundTruth = "file exists but is empty or trivial"
			return r, nil
		}
		r.GroundTruth = "file exists (" + formatSize(info.Size()) + " bytes)"
		r.Evidence = []string{path}
		return r, nil
	}

	return r, nil
}

func resolveClaimPath(path, cwd string) string {
	path = strings.TrimSpace(strings.Trim(path, `"'`+"`"))
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	if cwd != "" {
		return filepath.Clean(filepath.Join(cwd, path))
	}
	return filepath.Clean(path)
}

func formatSize(n int64) string {
	return strconv.FormatInt(n, 10)
}
