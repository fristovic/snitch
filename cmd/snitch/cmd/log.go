package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fristovic/snitch/internal/ipc"
	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/severity"
	"github.com/fristovic/snitch/internal/textutil"
	"github.com/spf13/cobra"
)

var (
	logRunID   string
	logTrace   bool
	logJSON    bool
	logHarness string
	logLimit   int
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show detailed verification for one agent run",
	Long: `Prints the full verification breakdown for a single run (--run), or lists
recent runs filtered by harness (--harness). Use snitch dashboard to browse history.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if logRunID == "" && logHarness == "" {
			return fmt.Errorf("either --run or --harness is required")
		}
		client, err := ipc.Connect(resolveSocket())
		if err != nil {
			daemonNotRunning()
			return nil
		}
		defer client.Close()

		if logRunID == "" {
			return printRunList(client, logHarness, logLimit)
		}

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

// printRunList shows recent runs for one harness (playbook 1.10: --harness filter).
func printRunList(client *ipc.Client, harness string, limit int) error {
	if limit <= 0 {
		limit = 20
	}
	data, err := client.Call("get_runs", map[string]any{"harness": harness, "limit": limit})
	if err != nil {
		return err
	}
	var runs []record.Run
	if err := json.Unmarshal(data, &runs); err != nil {
		return err
	}
	if len(runs) == 0 {
		fmt.Printf("no runs for harness %q\n", harness)
		return nil
	}
	for _, r := range runs {
		fmt.Printf("%s  %-6s  %s  %s\n", r.ID[:8], r.Verdict, r.CreatedAt.Format("2006-01-02 15:04"), textutil.TruncateRunes(formatPrompt(r.Command), 70))
	}
	return nil
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
		fmt.Printf("Prompt: %s\n", textutil.OneLine(formatPrompt(resp.Run.Command), 200))
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
		parts = append(parts, fmt.Sprintf("%q → %q", textutil.TruncateRunes(claimText(c), 80), textutil.TruncateRunes(actual, 80)))
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

func formatPrompt(s string) string {
	s = strings.ReplaceAll(s, "<user_query>", "")
	s = strings.ReplaceAll(s, "</user_query>", "")
	return strings.TrimSpace(s)
}

func init() {
	logCmd.Flags().StringVar(&logRunID, "run", "", "Run ID to show")
	logCmd.Flags().BoolVar(&logTrace, "trace", false, "Show verification pipeline trace")
	logCmd.Flags().BoolVar(&logJSON, "json", false, "Output run and claims as JSON")
	logCmd.Flags().StringVar(&logHarness, "harness", "", "List recent runs for one harness (cursor|claude|codex|pi|opencode)")
	logCmd.Flags().IntVar(&logLimit, "limit", 20, "Max runs to list with --harness")
}
