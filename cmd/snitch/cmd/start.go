package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Open Snitch Bar in the menu bar",
	Long: `Opens Snitch Bar (menu bar app, no Dock icon).

Claim verification is controlled from the menu bar:
  Start Snitching  — turn the verifier on
  Stop Snitching   — pause`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return openSnitchBar()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func printSnitchResumeHint(w io.Writer) {
	fmt.Fprintln(w, "To begin verifying claims:")
	fmt.Fprintln(w, "  snitch start")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Then choose Start Snitching in the menu bar.")
}

func daemonNotRunning() {
	fmt.Fprintln(os.Stderr, "Snitch is not verifying claims.")
	fmt.Fprintln(os.Stderr, "")
	printSnitchResumeHint(os.Stderr)
	os.Exit(1)
}

func printSnitchStoppedStatus() {
	fmt.Println("snitch: not snitching")
	fmt.Println()
	printSnitchResumeHint(os.Stdout)
}
