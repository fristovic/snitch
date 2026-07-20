//go:build stress

package stress

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fristovic/snitch/internal/record"
)

func TestStressMatrix(t *testing.T) {
	for _, sc := range AllCases() {
		t.Run(sc.Name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			projectDir := filepath.Join(dir, "project")
			if err := os.MkdirAll(projectDir, 0o755); err != nil {
				t.Fatal(err)
			}

			got, err := RunCase(sc, dir, projectDir)
			if err != nil {
				t.Fatal(err)
			}

			if got.MatchedFlagged != sc.ExpectFlagged {
				t.Errorf("case %q (%s %s): got flagged=%v want=%v verdict=%s",
					sc.Name, sc.ClaimType, sc.Category, got.MatchedFlagged, sc.ExpectFlagged, got.Verdict)
				if got.MatchedClaim != nil {
					t.Errorf("  claimed: %q actual: %q sev=%d", got.MatchedClaim.Claimed, got.MatchedClaim.Actual, got.MatchedClaim.Severity)
				}
				for _, c := range got.Claims {
					if record.IsContradictedClaim(c) && c.Severity >= 2 {
						t.Logf("  flagged claim: type=%s claimed=%q actual=%q", c.ClaimType, c.Claimed, c.Actual)
					}
				}
			}
		})
	}
}

func TestStressFamilyFileProse(t *testing.T) {
	runFamily(t, FileProseCases())
}

func TestStressFamilyStubNoAction(t *testing.T) {
	runFamily(t, StubNoActionCases())
}

func TestStressFamilyGit(t *testing.T) {
	runFamily(t, GitCases())
}

func TestStressFamilyShell(t *testing.T) {
	runFamily(t, ShellCases())
}

func TestStressFamilyConsistency(t *testing.T) {
	runFamily(t, ConsistencyCases())
}

func TestStressLiveSession(t *testing.T) {
	runFamily(t, LiveSessionCases())
}

func TestStressSession(t *testing.T) {
	for _, sc := range SessionCases() {
		t.Run(sc.Name, func(t *testing.T) {
			dir := t.TempDir()
			projectDir := filepath.Join(dir, "project")
			if err := os.MkdirAll(projectDir, 0o755); err != nil {
				t.Fatal(err)
			}
			results, err := RunSession(sc.Turns, dir, projectDir, "sess-"+sc.Name)
			if err != nil {
				t.Fatal(err)
			}
			if len(results) == 0 {
				t.Fatal("no results")
			}
			final := results[len(results)-1]
			if final.MatchedFlagged != sc.ExpectFinalFlagged {
				t.Errorf("%s: final flagged=%v want=%v verdict=%s",
					sc.Name, final.MatchedFlagged, sc.ExpectFinalFlagged, final.Verdict)
				if final.MatchedClaim != nil {
					t.Errorf("  claimed=%q actual=%q sev=%d", final.MatchedClaim.Claimed, final.MatchedClaim.Actual, final.MatchedClaim.Severity)
				}
			}
		})
	}
}

func runFamily(t *testing.T, cases []StressCase) {
	t.Helper()
	for _, sc := range cases {
		t.Run(sc.Name, func(t *testing.T) {
			dir := t.TempDir()
			projectDir := filepath.Join(dir, "project")
			if err := os.MkdirAll(projectDir, 0o755); err != nil {
				t.Fatal(err)
			}
			got, err := RunCase(sc, dir, projectDir)
			if err != nil {
				t.Fatal(err)
			}
			if got.MatchedFlagged != sc.ExpectFlagged {
				t.Errorf("%s: got flagged=%v want=%v", sc.Name, got.MatchedFlagged, sc.ExpectFlagged)
			}
		})
	}
}
