package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/fristovic/snitch/internal/ipc"
	"github.com/spf13/cobra"
)

// labelShareFlag mirrors the telemetry share opt-in for one-off labels.
var labelShareFlag bool

var labelCmd = &cobra.Command{
	Use:   "label [run-id] [correct|incorrect]",
	Short: "Mark a run's verdict correct or incorrect (trains Snitch)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID, verdict := args[0], args[1]
		if verdict != "correct" && verdict != "incorrect" {
			return fmt.Errorf("verdict must be 'correct' or 'incorrect'")
		}
		client, err := ipc.Connect(resolveSocket())
		if err != nil {
			return err
		}
		defer client.Close()
		resp, err := client.Call("set_label", map[string]any{
			"run_id": runID,
			"label":  verdict,
			"shared": labelShareFlag,
		})
		if err != nil {
			return err
		}
		var out struct {
			OK     bool `json:"ok"`
			Shared bool `json:"shared"`
		}
		_ = json.Unmarshal(resp, &out)
		share := "not shared"
		if out.Shared {
			share = "shared anonymously"
		}
		fmt.Printf("Thanks — this helps train Snitch. Verdict '%s' recorded for %s (%s).\n", verdict, runID, share)
		return nil
	},
}

var (
	missedRunID   string
	missedClaimed string
	missedActual  string
)

// labelMissedCmd reports a false negative: the agent made a false claim
// Snitch missed. These are the highest-value training examples.
var labelMissedCmd = &cobra.Command{
	Use:   "missed --claimed \"...\" --actual \"...\"",
	Short: "Report a claim Snitch missed (false negative)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if missedClaimed == "" || missedActual == "" {
			return fmt.Errorf("--claimed and --actual are required")
		}
		client, err := ipc.Connect(resolveSocket())
		if err != nil {
			return err
		}
		defer client.Close()
		resp, err := client.Call("add_missed_claim", map[string]any{
			"run_id":  missedRunID,
			"claimed": missedClaimed,
			"actual":  missedActual,
			"shared":  labelShareFlag,
		})
		if err != nil {
			return err
		}
		var out struct {
			OK     bool `json:"ok"`
			Shared bool `json:"shared"`
		}
		_ = json.Unmarshal(resp, &out)
		share := "not shared"
		if out.Shared {
			share = "metadata shared anonymously"
		}
		fmt.Printf("Missed claim recorded (%s). The claim text never leaves your machine.\n", share)
		return nil
	},
}

func init() {
	labelCmd.Flags().BoolVar(&labelShareFlag, "share", false, "Share this verdict anonymously to train Snitch")
	labelMissedCmd.Flags().StringVar(&missedRunID, "run", "", "Run ID this missed claim belongs to (optional)")
	labelMissedCmd.Flags().StringVar(&missedClaimed, "claimed", "", "What the agent said")
	labelMissedCmd.Flags().StringVar(&missedActual, "actual", "", "What actually happened")
	labelMissedCmd.Flags().BoolVar(&labelShareFlag, "share", false, "Share metadata anonymously to train Snitch")
	labelCmd.AddCommand(labelMissedCmd)
	rootCmd.AddCommand(labelCmd)
}
