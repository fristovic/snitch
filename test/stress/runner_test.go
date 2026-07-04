//go:build stress

package stress

import (
	"os"
	"path/filepath"
	"testing"
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

			if got.MatchedLie != sc.ExpectLie {
				t.Errorf("case %q (%s %s): got lie=%v want=%v verdict=%s",
					sc.Name, sc.LieType, sc.Category, got.MatchedLie, sc.ExpectLie, got.Verdict)
				if got.LieClaim != nil {
					t.Errorf("  claimed: %q actual: %q sev=%d", got.LieClaim.Claimed, got.LieClaim.Actual, got.LieClaim.Severity)
				}
				for _, c := range got.Claims {
					if c.Verified < 0 && c.Severity >= 2 {
						t.Logf("  lie claim: type=%s claimed=%q actual=%q", c.ClaimType, c.Claimed, c.Actual)
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
			if final.MatchedLie != sc.ExpectFinalLie {
				t.Errorf("%s: final lie=%v want=%v verdict=%s",
					sc.Name, final.MatchedLie, sc.ExpectFinalLie, final.Verdict)
				if final.LieClaim != nil {
					t.Errorf("  claimed=%q actual=%q sev=%d", final.LieClaim.Claimed, final.LieClaim.Actual, final.LieClaim.Severity)
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
			if got.MatchedLie != sc.ExpectLie {
				t.Errorf("%s: got lie=%v want=%v", sc.Name, got.MatchedLie, sc.ExpectLie)
			}
		})
	}
}
