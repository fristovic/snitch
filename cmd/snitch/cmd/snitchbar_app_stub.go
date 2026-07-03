//go:build !darwin

package cmd

import "fmt"

func openSnitchBar() error {
	return fmt.Errorf("snitch start requires macOS")
}

func findSnitchBarApp() (string, error) {
	return "", fmt.Errorf("snitch start requires macOS")
}
