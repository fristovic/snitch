package platform

import (
	"runtime"
	"testing"
)

func TestShellCommand(t *testing.T) {
	cmd := ShellCommand("echo hello")
	if cmd == nil {
		t.Fatal("nil cmd")
	}
	path := cmd.Path
	args := cmd.Args

	switch runtime.GOOS {
	case "windows":
		if path != "cmd.exe" && path != `C:\Windows\system32\cmd.exe` {
			// Path may be fully qualified on Windows.
			if len(args) < 2 || args[len(args)-2] != "/C" {
				t.Fatalf("expected cmd.exe /C, got path=%s args=%v", path, args)
			}
		}
	default:
		if len(args) < 3 || args[1] != "-c" || args[2] != "echo hello" {
			t.Fatalf("expected sh -c, got %v", args)
		}
	}
}
