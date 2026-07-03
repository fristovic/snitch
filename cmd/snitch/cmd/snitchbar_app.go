//go:build darwin

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func openSnitchBar() error {
	app, err := findSnitchBarApp()
	if err != nil {
		return err
	}
	if err := exec.Command("open", app).Run(); err != nil {
		return fmt.Errorf("open Snitch Bar: %w", err)
	}
	fmt.Println("Opened Snitch Bar — choose Start Snitching in the menu bar to detect lies.")
	return nil
}

func findSnitchBarApp() (string, error) {
	if override := os.Getenv("SNITCH_BAR_APP"); override != "" {
		if snitchBarBundleValid(override) {
			return override, nil
		}
		return "", fmt.Errorf("SNITCH_BAR_APP is not a valid Snitch Bar.app bundle: %s", override)
	}

	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local", "share", "snitch", "Snitch Bar.app"),
	}

	if prefix, err := brewPrefix("snitch"); err == nil && prefix != "" {
		candidates = append(candidates, filepath.Join(prefix, "Snitch Bar.app"))
	}
	if prefix, err := brewPrefix(""); err == nil && prefix != "" {
		candidates = append(candidates, filepath.Join(prefix, "opt", "snitch", "Snitch Bar.app"))
	}

	for _, app := range candidates {
		if snitchBarBundleValid(app) {
			return app, nil
		}
	}
	return "", fmt.Errorf("cannot find snitch bar app — reinstall Snitch or set SNITCH_BAR_APP")
}

func snitchBarBundleValid(app string) bool {
	return fileExists(filepath.Join(app, "Contents", "MacOS", "snitchbar"))
}

func brewPrefix(formula string) (string, error) {
	args := []string{"--prefix"}
	if formula != "" {
		args = append(args, formula)
	}
	out, err := exec.Command("brew", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
