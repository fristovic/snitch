package notify

import (
	"fmt"
	"path/filepath"

	"github.com/fristovic/snitch/internal/claims"
)

// FormatNotificationFields formats a notification using sentence/context when present.
func FormatNotificationFields(d claims.DisplayFields, projectPath string) (title, body string) {
	label := claims.ClaimTypeLabel(d.ClaimType)
	title = "Snitch — " + label
	if projectPath != "" {
		base := filepath.Base(projectPath)
		if base != "" && base != "." {
			title = fmt.Sprintf("Snitch — %s (%s)", label, base)
		}
	}
	body = claims.NotificationBody(d, 120)
	return title, body
}
