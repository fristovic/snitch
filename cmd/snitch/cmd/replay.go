package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fristovic/snitch/internal/capture"
	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/transcript"
	"github.com/fristovic/snitch/internal/verify"
)

var (
	replayHarness  string
	replayLiesOnly bool
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
				claims, _ := store.GetClaimsByRun(turn.RunID)
				totalTurns++
				totalClaims += len(claims)
				printed := false
				for _, c := range claims {
					isLie := c.Verified < 0
					if isLie {
						flagged++
					}
					if replayLiesOnly && !isLie {
						continue
					}
					if !printed {
						fmt.Printf("\n%s  turn %s  (%s)\n", path, turn.RunID, harness)
						printed = true
					}
					mark := "OK "
					if isLie {
						mark = "LIE"
					}
					fmt.Printf("  [%s] %-18s sev=%d  %q\n", mark, c.ClaimType, c.Severity, truncateStr(c.Claimed, 90))
					if isLie && c.Actual != "" {
						fmt.Printf("        actual: %s\n", truncateStr(c.Actual, 90))
					}
				}
			}
		}

		fmt.Printf("\nreplayed %d transcript(s): %d turns, %d claims, %d flagged as lies\n",
			len(paths), totalTurns, totalClaims, flagged)
		fmt.Println("review every LIE above — each false positive is a pattern to fix or label.")
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
	replayCmd.Flags().BoolVar(&replayLiesOnly, "lies-only", false, "Print only claims flagged as lies")
	rootCmd.AddCommand(replayCmd)
}
