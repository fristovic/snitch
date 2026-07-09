//go:build darwin

package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindSnitchBarApp(t *testing.T) {
	tests := []struct {
		name      string
		checkValid bool
	}{
		{name: "override"},
		{name: "doctor_bundle_valid", checkValid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			if tt.checkValid && !snitchBarBundleValid(got) {
				t.Fatalf("doctor would fail Snitch Bar.app check for %q", got)
			}
		})
	}
}
