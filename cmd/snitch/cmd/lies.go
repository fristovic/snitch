package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/fristovic/snitch/internal/ipc"
	"github.com/fristovic/snitch/internal/record"
	"github.com/spf13/cobra"
)

var (
	liesType    string
	liesProject string
	liesSession string
	liesSince   string
	liesLimit   int
	liesJSON    bool
)

var liesCmd = &cobra.Command{
	Use:   "lies",
	Short: "List caught lies from assistant prose",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ipc.Connect(resolveSocket())
		if err != nil {
			daemonNotRunning()
			return nil
		}
		defer client.Close()

		params := map[string]any{
			"lies_only": true,
			"limit":     liesLimit,
		}
		if liesType != "" {
			params["claim_type"] = liesType
		}
		if liesProject != "" {
			params["project_path"] = liesProject
		}
		if liesSession != "" {
			params["session_id"] = liesSession
		}
		if liesSince != "" {
			if t, err := parseSince(liesSince); err == nil {
				params["since"] = t.Format(time.RFC3339)
			} else {
				return fmt.Errorf("invalid --since: %w", err)
			}
		}

		data, err := client.Call("get_claims", params)
		if err != nil {
			return err
		}
		var claims []record.LieClaim
		if err := json.Unmarshal(data, &claims); err != nil {
			return err
		}
		if liesJSON {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(claims)
		}
		if len(claims) == 0 {
			fmt.Println("no lies caught — snitch is quiet")
			return nil
		}
		for _, c := range claims {
			proj := truncateLog(c.ProjectPath, 30)
			fmt.Printf("%s  %-12s  %-12s  %q  →  %s\n",
				c.RunCreated.Format("15:04:05"),
				proj,
				c.ClaimType,
				truncateLog(c.Claimed, 60),
				c.Actual,
			)
		}
		return nil
	},
}

func parseSince(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-d), nil
	}
	return time.Time{}, fmt.Errorf("use RFC3339 or duration like 24h")
}

func init() {
	liesCmd.Flags().StringVar(&liesType, "type", "", "Filter by claim type (test_pass, committed, ...)")
	liesCmd.Flags().StringVar(&liesProject, "project", "", "Filter by project path")
	liesCmd.Flags().StringVar(&liesSession, "session", "", "Filter by session ID")
	liesCmd.Flags().StringVar(&liesSince, "since", "", "Only lies since time (RFC3339 or duration)")
	liesCmd.Flags().IntVar(&liesLimit, "limit", 50, "Max lies to show")
	liesCmd.Flags().BoolVar(&liesJSON, "json", false, "Output JSON")
}
