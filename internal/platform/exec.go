//go:build !windows

package platform

import "os/exec"

// ShellCommand returns an *exec.Cmd that runs a user-provided shell line on Unix.
func ShellCommand(line string) *exec.Cmd {
	return exec.Command("sh", "-c", line)
}
