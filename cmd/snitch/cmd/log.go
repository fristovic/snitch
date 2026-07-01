package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/ipc"
	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/severity"
	"github.com/spf13/cobra"
)

var (
	logRunID      string
	logTrace      bool
	logWatch      bool
	logLimit      int
	logShowAll    bool
	logProject    string
	logType       string
	logSince      string
	logSearch     string
	logJSON       bool
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show agent runs that failed verification",
	Long:  "By default shows only failed/warned runs with a short claimed→actual summary. Use --all to include passes.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ipc.Connect(resolveSocket())
		if err != nil {
			daemonNotRunning()
			return nil
		}
		defer client.Close()

		if logRunID != "" {
			return printRunDetail(client, logRunID)
		}

		params := map[string]any{"limit": logLimit}
		if !logShowAll {
			params["failures_only"] = true
		}
		if logProject != "" {
			params["project_path"] = logProject
		}
		if logSearch != "" {
			params["search"] = logSearch
		}
		if logSince != "" {
			if t, err := parseSince(logSince); err == nil {
				params["since"] = t.Format(time.RFC3339)
			} else {
				return fmt.Errorf("invalid --since: %w", err)
			}
		}

		data, err := client.Call("get_runs", params)
		if err != nil {
			return err
		}
		var runs []record.Run
		if err := json.Unmarshal(data, &runs); err != nil {
			return err
		}
		if logJSON {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(runs)
		}
		if len(runs) == 0 {
			if logShowAll {
				fmt.Println("no runs recorded yet")
			} else {
				fmt.Println("no failed runs — snitch is quiet")
			}
		}
		for _, r := range runs {
			if logType != "" {
				if err := printRunSummaryFiltered(client, r, logType); err != nil {
					return err
				}
			} else if err := printRunSummary(client, r); err != nil {
				return err
			}
		}
		if logWatch {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			fmt.Println("watching for failed runs... (Ctrl+C to stop)")
			return ipc.Watch(ctx, resolveSocket(), func(msg ipc.EventMsg) error {
				if msg.Event != "run.completed" {
					return nil
				}
				var p event.RunVerifiedPayload
				if err := json.Unmarshal(msg.Data, &p); err != nil {
					return nil
				}
				if !isSnitchWorthy(p.Verdict) {
					return nil
				}
				return printRunSummary(client, record.Run{
					ID: p.RunID, Verdict: p.Verdict,
				})
			})
		}
		return nil
	},
}

func isSnitchWorthy(v record.Verdict) bool {
	return v == record.VerdictFail || v == record.VerdictWarn
}

func printRunSummaryFiltered(client *ipc.Client, r record.Run, claimType string) error {
	claims, err := fetchClaims(client, r.ID)
	if err != nil {
		return printRunSummary(client, r)
	}
	for _, c := range claims {
		if c.ClaimType == claimType && (c.Severity >= int(severity.Level2) || c.Verified < 0) {
			line := fmt.Sprintf("%s  %s  %s", r.ID[:8], r.Verdict, c.ClaimType)
			line += fmt.Sprintf("  %q → %q", truncateLog(c.Claimed, 60), truncateLog(c.Actual, 60))
			fmt.Println(line)
		}
	}
	return nil
}

func printRunSummary(client *ipc.Client, r record.Run) error {
	claims, err := fetchClaims(client, r.ID)
	if err == nil && r.Command == "" {
		if detail, err := fetchRun(client, r.ID); err == nil {
			r = detail
		}
	}
	summary := ""
	if r.FalseClaims > 0 || isSnitchWorthy(r.Verdict) {
		if err == nil {
			summary = failureSummary(claims)
		}
	}
	line := fmt.Sprintf("%s  %s", r.ID[:8], r.Verdict)
	if r.Command != "" {
		line += fmt.Sprintf("  %q", truncateLog(r.Command, 60))
	}
	if summary != "" {
		line += "  " + summary
	}
	fmt.Println(line)
	return nil
}

func fetchRun(client *ipc.Client, runID string) (record.Run, error) {
	data, err := client.Call("get_run", map[string]string{"id": runID})
	if err != nil {
		return record.Run{}, err
	}
	var resp struct {
		Run record.Run `json:"run"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return record.Run{}, err
	}
	return resp.Run, nil
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
		fmt.Printf("Prompt: %s\n", truncateLog(resp.Run.Command, 200))
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

func fetchClaims(client *ipc.Client, runID string) ([]record.Claim, error) {
	data, err := client.Call("get_run", map[string]string{"id": runID})
	if err != nil {
		return nil, err
	}
	var resp struct {
		Claims []record.Claim `json:"claims"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Claims, nil
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

func init() {
	logCmd.Flags().StringVar(&logRunID, "run", "", "Show detailed verification for a run")
	logCmd.Flags().BoolVar(&logTrace, "trace", false, "Show verification pipeline trace")
	logCmd.Flags().BoolVar(&logWatch, "watch", false, "Follow new failed runs")
	logCmd.Flags().IntVar(&logLimit, "limit", 10, "Number of runs to show")
	logCmd.Flags().BoolVar(&logShowAll, "all", false, "Include passing runs")
	logCmd.Flags().StringVar(&logProject, "project", "", "Filter runs by project path")
	logCmd.Flags().StringVar(&logType, "type", "", "Filter claims by claim type")
	logCmd.Flags().StringVar(&logSince, "since", "", "Only runs since time (RFC3339 or duration)")
	logCmd.Flags().StringVar(&logSearch, "search", "", "Search command text")
	logCmd.Flags().BoolVar(&logJSON, "json", false, "Output JSON")
}
