package main

import (
	"os"

	"github.com/fristovic/snitch/cmd/snitch/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
