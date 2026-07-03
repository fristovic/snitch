package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/fristovic/snitch/internal/ipc"
	"github.com/fristovic/snitch/internal/record"
	"github.com/spf13/cobra"
)

var statusDetailed bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show lie detection status",
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

		if st.TotalRuns == 0 {
			fmt.Println("\nSnitching — trigger a Cursor agent turn to see results")
		}

		if !statusDetailed {
			return nil
		}

		fmt.Printf("snitched runs: %d\n", st.SnitchedRuns)
		if st.TopLieType != "" {
			fmt.Printf("top lie type: %s\n", st.TopLieType)
		}
		if len(st.LieStats.ByClaimType) > 0 {
			fmt.Println("lies by type:")
			for t, n := range st.LieStats.ByClaimType {
				fmt.Printf("  %s: %d\n", t, n)
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
	statusCmd.Flags().BoolVar(&statusDetailed, "detailed", false, "Show lie statistics and recent failures")
}

