package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var socketPath string

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:   "snitch",
	Short: "Snitch agent verification CLI",
}

// Execute runs the CLI.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&socketPath, "socket", "", "IPC socket path (auto-detected if empty)")
	rootCmd.AddCommand(statusCmd, logCmd, liesCmd, configCmd, analyticsCmd, dashboardCmd)
}

func daemonNotRunning() {
	fmt.Fprintln(os.Stderr, "lie detector is not running. Open Snitch Bar from the menu bar, or choose Start Snitching from its menu.")
	os.Exit(1)
}
