//go:build darwin

package notify

import (
	"fmt"
	"os/exec"
)

// Notify shows a macOS Notification Center alert.
func Notify(title, body string) error {
	script := fmt.Sprintf(
		`display notification %q with title %q`,
		body, title,
	)
	return exec.Command("osascript", "-e", script).Run()
}
