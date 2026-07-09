package notify

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/textutil"
)

// FormatNotification returns the macOS notification title and body for a lie.
func FormatNotification(claimType, claimed, actual, projectPath string) (title, body string) {
	title = "Snitch — " + claimType
	if projectPath != "" {
		base := filepath.Base(projectPath)
		if base != "" && base != "." {
			title = fmt.Sprintf("Snitch — %s (%s)", claimType, base)
		}
	}
	claimed = strings.TrimSpace(claimed)
	actual = strings.TrimSpace(actual)
	if actual == "" {
		body = claimed
	} else {
		body = fmt.Sprintf("%q → %s", textutil.TruncateRunes(claimed, 120), textutil.TruncateRunes(actual, 120))
	}
	return title, body
}

// TopLieClaim picks the highest-severity false claim from a run.
func TopLieClaim(claims []record.Claim) *record.Claim {
	var best *record.Claim
	for i := range claims {
		c := &claims[i]
		if c.Verified != -1 {
			continue
		}
		if best == nil || c.Severity > best.Severity {
			best = c
		}
	}
	return best
}
