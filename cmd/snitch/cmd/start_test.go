//go:build darwin

package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindSnitchBarAppOverride(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "Snitch Bar.app")
	if err := os.MkdirAll(filepath.Join(app, "Contents", "MacOS"), 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(app, "Contents", "MacOS", "snitchbar")
	if err := os.WriteFile(bin, []byte{0}, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SNITCH_BAR_APP", app)
	got, err := findSnitchBarApp()
	if err != nil {
		t.Fatal(err)
	}
	if got != app {
		t.Fatalf("got %q want %q", got, app)
	}
}
