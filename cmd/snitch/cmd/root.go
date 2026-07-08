package cmd

import (
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
	rootCmd.AddCommand(statusCmd, logCmd, configCmd, analyticsCmd, dashboardCmd)
}
