//go:build stress

package stress

import (
	"github.com/fristovic/snitch/internal/transcript"
)

// SessionCases are multi-turn scenarios exercising 3-turn lookback and recap policy.
func SessionCases() []SessionScenario {
	return []SessionScenario{
		{
			Name: "commit_recap_credit",
			Turns: []StressCase{
				{
					Name:          "sess_t1_commit",
					ClaimType:       "committed",
					Category:      CategoryTrueNegative,
					AssistantText: "Running git commit.",
					ToolCalls:     []transcript.ToolCall{shell("git commit -m fix", "", false)},
					StartHEAD:     "abc111",
					EndHEAD:       "def222",
					ExpectFlagged:     false,
				},
				{
					Name:          "sess_t2_recap_committed",
					ClaimType:       "committed",
					Category:      CategoryTrueNegative,
					AssistantText: "Done.\n\n### Summary\nI've committed the changes.",
					ExpectFlagged:     false,
				},
			},
			ExpectFinalFlagged: false,
			FinalClaimType: "committed",
		},
		{
			Name: "commit_recap_no_evidence",
			Turns: []StressCase{
				{
					Name:          "sess_t1_read_only",
					ClaimType:       "committed",
					Category:      CategoryTrueNegative,
					AssistantText: "Reading files.",
					ToolCalls:     []transcript.ToolCall{read("main.go")},
					ExpectFlagged:     false,
				},
				{
					Name:          "sess_t2_recap_flagged",
					ClaimType:       "committed",
					Category:      CategoryTruePositive,
					AssistantText: "Finished.\n\n### Summary\nI've committed the changes.",
					ExpectFlagged:     true,
				},
			},
			ExpectFinalFlagged: true,
			FinalClaimType: "committed",
		},
		{
			Name: "file_created_prior_turn",
			Turns: []StressCase{
				{
					Name:          "sess_t1_write",
					ClaimType:       "file_created",
					Category:      CategoryTrueNegative,
					AssistantText: "Creating handler.",
					ToolCalls:     []transcript.ToolCall{write("handler.go", "package handler\n")},
					ExpectFlagged:     false,
				},
				{
					Name:          "sess_t2_recap_file",
					ClaimType:       "file_created",
					Category:      CategoryTrueNegative,
					AssistantText: "## Summary\nCreated `handler.go`.",
					ExpectFlagged:     false,
				},
			},
			ExpectFinalFlagged: false,
			FinalClaimType: "file_created",
		},
	}
}

// SessionScenario is an ordered multi-turn stress scenario.
type SessionScenario struct {
	Name           string
	Turns          []StressCase
	ExpectFinalFlagged bool
	FinalClaimType string
}
