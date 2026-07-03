//go:build !darwin

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "snitchbar requires macOS")
	os.Exit(1)
}
