package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fristovic/snitch/internal/claims"
	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/ipc"
	"github.com/fristovic/snitch/internal/record"
	"github.com/spf13/cobra"
)

var statusDetailed bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show claim verification status",
	RunE: func(cmd *cobra.Command, args []string) error {
		sock := ipc.ResolveSocket(socketPath)
		client, err := ipc.Connect(sock)
		if err != nil {
			printSnitchStoppedStatus()
			return nil
		}
		defer client.Close()

		data, err := client.Call("status", nil)
		if err != nil {
			return err
		}
		var st record.DaemonStatus
		if err := json.Unmarshal(data, &st); err != nil {
			return err
		}
		fmt.Printf("snitch: snitching\n")
		fmt.Printf("version: %s\n", st.Version)
		fmt.Printf("uptime: %ds\n", st.UptimeSeconds)
		fmt.Printf("total runs: %d\n", st.TotalRuns)
		fmt.Printf("projects watched: %d\n", st.ProjectsWatched)
		fmt.Printf("sessions seen: %d\n", st.SessionsSeen)

		// Show which harnesses are enabled.
		cfgData, cfgErr := client.Call("get_config", nil)
		if cfgErr == nil {
			var cfg config.Config
			if json.Unmarshal(cfgData, &cfg) == nil {
				var enabled []string
				for _, name := range config.HarnessNames() {
					if pc, ok := cfg.Platforms.ForHarness(name); ok && pc.Enabled {
						enabled = append(enabled, name)
					}
				}
				if len(enabled) > 0 {
					fmt.Printf("harnesses: %s\n", strings.Join(enabled, ", "))
				}
			}
		}

		if st.TotalRuns == 0 {
			fmt.Println("\nSnitching — trigger an agent turn to see results")
		}

		if !statusDetailed {
			return nil
		}

		fmt.Printf("snitched runs: %d\n", st.SnitchedRuns)
		if len(st.RunsByHarness) > 0 {
			fmt.Println("runs by harness:")
			for h, n := range st.RunsByHarness {
				fmt.Printf("  %s: %d\n", h, n)
			}
		}
		if st.MostCommonFalseClaimType != "" {
			fmt.Printf("most common false claim: %s (%s)\n", claims.ClaimTypeLabel(st.MostCommonFalseClaimType), st.MostCommonFalseClaimType)
		}
		if len(st.ClaimStats.ByClaimType) > 0 {
			fmt.Println("false claims by type:")
			for t, n := range st.ClaimStats.ByClaimType {
				fmt.Printf("  %s (%s): %d\n", claims.ClaimTypeLabel(t), t, n)
			}
		}

		runs, err := client.Call("get_runs", map[string]any{"limit": 5, "failures_only": true})
		if err == nil {
			var recent []record.Run
			_ = json.Unmarshal(runs, &recent)
			if len(recent) > 0 {
				fmt.Println("\nrecent failures:")
				for _, r := range recent {
					fmt.Printf("  %s  %s  %s\n", r.ID[:8], r.Verdict, r.CreatedAt.Format("15:04:05"))
				}
			}
		}
		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusDetailed, "detailed", false, "Show claim statistics and recent failures")
}
