package cmd

import (
	"fmt"

	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/platform"
	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/report"
	"github.com/fristovic/snitch/internal/version"
	"github.com/spf13/cobra"
)

var analyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "Show what would be reported (dry-run)",
	RunE: func(cmd *cobra.Command, args []string) error {
		paths, _ := platform.Resolve()
		cfg, _ := config.Load(paths.ConfigPath)
		store, err := record.Open(paths.DataDir)
		if err != nil {
			return err
		}
		defer store.Close()
		reporter := report.New(cfg.Analytics, store, version.Version)
		data, err := reporter.DryRun()
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	},
}
