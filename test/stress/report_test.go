//go:build stress

package stress

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWriteReports generates markdown reports (run with STRESS_WRITE_REPORTS=1).
func TestWriteReports(t *testing.T) {
	if os.Getenv("STRESS_WRITE_REPORTS") != "1" {
		t.Skip("set STRESS_WRITE_REPORTS=1 to regenerate reports")
	}
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "project")
	_ = os.MkdirAll(projectDir, 0o755)

	families := []struct {
		name  string
		file  string
		cases func() []StressCase
	}{
		{"file_prose", "file_prose.md", FileProseCases},
		{"stub_noaction", "stub_noaction.md", StubNoActionCases},
		{"git", "git.md", GitCases},
		{"shell", "shell.md", ShellCases},
		{"consistency", "consistency.md", ConsistencyCases},
		{"live_session", "live_session.md", LiveSessionCases},
	}

	reportsDir := filepath.Join("reports")
	_ = os.MkdirAll(reportsDir, 0o755)

	for _, fam := range families {
		var fp, fn, tp, tn []string
		for _, sc := range fam.cases() {
			caseDir := filepath.Join(dir, fam.name, sc.Name)
			_ = os.MkdirAll(caseDir, 0o755)
			caseProject := filepath.Join(caseDir, "project")
			_ = os.MkdirAll(caseProject, 0o755)
			got, err := RunCase(sc, caseDir, caseProject)
			if err != nil {
				t.Fatal(err)
			}
			match := got.MatchedFlagged == sc.ExpectFlagged
			line := formatReportLine(sc, got, match)
			switch sc.Category {
			case CategoryFalsePositive:
				fp = append(fp, line)
			case CategoryFalseNegative:
				fn = append(fn, line)
			case CategoryTruePositive:
				tp = append(tp, line)
			case CategoryTrueNegative:
				tn = append(tn, line)
			}
		}
		header := "| Case | Status | Expect flagged | Prose | Actual | Root cause |\n|------|--------|------------|-------|--------|------------|\n"
		body := fmt.Sprintf("# Stress report: %s\n\n## False positives\n\n%s%s\n\n## False negatives\n\n%s%s\n\n## True positives\n\n%s%s\n\n## True negatives\n\n%s%s\n",
			fam.name,
			header, strings.Join(fp, "\n"),
			header, strings.Join(fn, "\n"),
			header, strings.Join(tp, "\n"),
			header, strings.Join(tn, "\n"))
		if err := os.WriteFile(filepath.Join(reportsDir, fam.file), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func formatReportLine(sc StressCase, got CaseResult, match bool) string {
	status := "pass"
	if !match {
		status = "**MISMATCH**"
	}
	actual := "not flagged"
	if got.MatchedFlagged {
		actual = "flagged"
		if got.MatchedClaim != nil {
			actual = fmt.Sprintf("flagged: %q → %q", got.MatchedClaim.Claimed, got.MatchedClaim.Actual)
		}
	}
	snippet := sc.AssistantText
	if len(snippet) > 60 {
		snippet = snippet[:57] + "..."
	}
	return fmt.Sprintf("| %s | %s | expect_flagged=%v | %s | %s | %s |",
		sc.Name, status, sc.ExpectFlagged, snippet, actual, sc.Notes)
}
