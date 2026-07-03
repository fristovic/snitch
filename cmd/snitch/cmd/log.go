package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fristovic/snitch/internal/ipc"
	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/severity"
	"github.com/spf13/cobra"
)

var (
	logRunID string
	logTrace bool
	logJSON  bool
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show detailed verification for one agent run",
	Long:  "Prints the full verification breakdown for a single run. Use snitch dashboard to browse history.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ipc.Connect(resolveSocket())
		if err != nil {
			daemonNotRunning()
			return nil
		}
		defer client.Close()

		if logJSON {
			data, err := client.Call("get_run", map[string]string{"id": logRunID})
			if err != nil {
				return err
			}
			var resp struct {
				Run    record.Run     `json:"run"`
				Claims []record.Claim `json:"claims"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(resp)
		}
		return printRunDetail(client, logRunID)
	},
}

func printRunDetail(client *ipc.Client, runID string) error {
	data, err := client.Call("get_run", map[string]string{"id": runID})
	if err != nil {
		return err
	}
	var resp struct {
		Run    record.Run     `json:"run"`
		Claims []record.Claim `json:"claims"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return err
	}
	fmt.Printf("Run: %s\nVerdict: %s\n", resp.Run.ID, resp.Run.Verdict)
	if resp.Run.SessionID != "" {
		fmt.Printf("Session: %s\n", resp.Run.SessionID)
	}
	if resp.Run.ProjectPath != "" {
		fmt.Printf("Project: %s\n", resp.Run.ProjectPath)
	}
	fmt.Printf("Tool calls: %d\n", resp.Run.ToolCallCount)
	if resp.Run.Command != "" {
		fmt.Printf("Prompt: %s\n", truncateLog(formatPrompt(resp.Run.Command), 200))
	}
	if resp.Run.Harness != "" {
		fmt.Printf("Harness: %s\n", resp.Run.Harness)
	}
	if logTrace && len(resp.Run.Trace) > 0 {
		fmt.Println("\nTrace:")
		for _, line := range resp.Run.Trace {
			fmt.Println(" ", line)
		}
	}
	if summary := failureSummary(resp.Claims); summary != "" {
		fmt.Println("Summary:", summary)
	}
	for _, c := range resp.Claims {
		if c.Severity < int(severity.Level2) && c.Verified > 0 {
			continue
		}
		label := c.ClaimType
		if c.Source == "prose" {
			label = c.ClaimType + " (prose)"
		}
		fmt.Printf("  [%s] %s → %s (sev %d, %s)\n", label, claimText(c), c.Actual, c.Severity, c.Verifier)
	}
	return nil
}

func failureSummary(claims []record.Claim) string {
	var parts []string
	for _, c := range claims {
		if c.Severity < int(severity.Level2) && c.Verified > 0 {
			continue
		}
		actual := c.Actual
		if actual == "" {
			actual = "(not done)"
		}
		parts = append(parts, fmt.Sprintf("%q → %q", truncateLog(claimText(c), 80), truncateLog(actual, 80)))
	}
	return strings.Join(parts, "; ")
}

func claimText(c record.Claim) string {
	if c.Claimed != "" {
		return c.Claimed
	}
	if c.Target != "" {
		return c.ClaimType + " " + c.Target
	}
	return c.ClaimType
}

func truncateLog(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func formatPrompt(s string) string {
	s = strings.ReplaceAll(s, "<user_query>", "")
	s = strings.ReplaceAll(s, "</user_query>", "")
	return strings.TrimSpace(s)
}

func init() {
	logCmd.Flags().StringVar(&logRunID, "run", "", "Run ID to show (required)")
	logCmd.Flags().BoolVar(&logTrace, "trace", false, "Show verification pipeline trace")
	logCmd.Flags().BoolVar(&logJSON, "json", false, "Output run and claims as JSON")
	_ = logCmd.MarkFlagRequired("run")
}
