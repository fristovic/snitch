//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func copyToClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func openTerminal(command string) error {
	script := fmt.Sprintf(`tell application "Terminal"
	activate
	do script %q
end tell`, command)
	return exec.Command("osascript", "-e", script).Run()
}

func openPath(path string) error {
	return exec.Command("open", path).Run()
}
