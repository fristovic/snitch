package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fristovic/snitch/internal/capture"
	"github.com/fristovic/snitch/internal/claims"
	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/transcript"
	"github.com/fristovic/snitch/internal/verify"
)

var (
	replayHarness  string
	replayFlaggedOnly bool
)

// replayCmd runs on-device transcripts through the full parse→verify pipeline
// offline, without a daemon. It is the accuracy-measurement instrument: run it
// over your real session history and inspect every claim Snitch would flag.
var replayCmd = &cobra.Command{
	Use:   "replay <transcript.jsonl | directory>",
	Short: "Replay transcripts through the verification pipeline offline",
	Long: `Replay parses one transcript (or every transcript under a directory),
assembles turns, and runs the full verification pipeline against a throwaway
database. Nothing touches the daemon or your real Snitch data.

Use it to measure false positives on your own sessions, or to validate a new
harness parser before opening a PR.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		paths, err := collectTranscripts(args[0])
		if err != nil {
			return err
		}
		if len(paths) == 0 {
			return fmt.Errorf("no .jsonl transcripts found under %s", args[0])
		}

		// Throwaway store + synchronous verify engine.
		tmp, err := os.MkdirTemp("", "snitch-replay-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		store, err := record.Open(tmp)
		if err != nil {
			return err
		}
		defer store.Close()
		bus := event.NewBus()
		defer bus.Close()
		engine := verify.NewEngine(bus, store, config.Default().Verification, "replay", nil)

		var totalTurns, totalClaims, flagged int
		for _, path := range paths {
			harness := replayHarness
			if harness == "" {
				harness = transcript.GuessHarness(path)
			}
			parser, resolver, ok := transcript.ParserFor(harness)
			if !ok {
				fmt.Fprintf(os.Stderr, "skip %s: unknown harness (use --harness)\n", path)
				continue
			}
			turns, err := transcript.ReplayTurns(parser, resolver, path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "skip %s: %v\n", path, err)
				continue
			}
			for _, turn := range turns {
				turn.Harness = harness
				engine.VerifyPayload(capture.BuildRunPayload(turn))
				runClaims, _ := store.GetClaimsByRun(turn.RunID)
				totalTurns++
				totalClaims += len(runClaims)
				printed := false
				for _, c := range runClaims {
					isFlagged := c.Verified < 0
					if isFlagged {
						flagged++
					}
					if replayFlaggedOnly && !isFlagged {
						continue
					}
					if !printed {
						fmt.Printf("\n%s  turn %s  (%s)\n", path, turn.RunID, harness)
						printed = true
					}
					mark := "OK "
					if isFlagged {
						mark = "FLAG"
					}
					fmt.Printf("  [%s] %-22s sev=%d  %q\n", mark, claims.ClaimTypeLabel(c.ClaimType), c.Severity, truncateStr(claims.FlaggedText(claims.FromRecord(c)), 90))
					if isFlagged && c.Actual != "" {
						fmt.Printf("        checked: %s\n", truncateStr(c.Actual, 90))
					}
				}
			}
		}

		fmt.Printf("\nreplayed %d transcript(s): %d turns, %d claims, %d flagged as false claims\n",
			len(paths), totalTurns, totalClaims, flagged)
		fmt.Println("review every FLAG above — each false positive is a pattern to fix or label.")
		return nil
	},
}

func collectTranscripts(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{root}, nil
	}
	var out []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(path, ".jsonl") {
			out = append(out, path)
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

func truncateStr(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func init() {
	replayCmd.Flags().StringVar(&replayHarness, "harness", "", "Force harness (cursor|claude|codex|pi); auto-detected from path if empty")
	replayCmd.Flags().BoolVar(&replayFlaggedOnly, "false-claims-only", false, "Print only claims flagged as false")
	// Alias kept so scripts from ≤0.3.x keep working after upgrade.
	replayCmd.Flags().BoolVar(&replayFlaggedOnly, "lies-only", false, "Deprecated alias for --false-claims-only")
	_ = replayCmd.Flags().MarkHidden("lies-only")
	rootCmd.AddCommand(replayCmd)
}
