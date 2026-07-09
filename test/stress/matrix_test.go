//go:build stress

package stress

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestWriteMatrix generates docs/stress-test-matrix.md (run with STRESS_WRITE_REPORTS=1).
func TestWriteMatrix(t *testing.T) {
	if os.Getenv("STRESS_WRITE_REPORTS") != "1" {
		t.Skip("set STRESS_WRITE_REPORTS=1 to regenerate matrix")
	}
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "project")
	_ = os.MkdirAll(projectDir, 0o755)

	type row struct {
		claimType        string
		tp, tn, fp, fn int
	}
	byType := map[string]*row{}

	for _, sc := range AllCases() {
		caseDir := filepath.Join(dir, sc.Name)
		_ = os.MkdirAll(caseDir, 0o755)
		caseProject := filepath.Join(caseDir, "project")
		_ = os.MkdirAll(caseProject, 0o755)

		got, err := RunCase(sc, caseDir, caseProject)
		if err != nil {
			t.Fatal(err)
		}
		lt := sc.ClaimType
		if byType[lt] == nil {
			byType[lt] = &row{claimType: lt}
		}
		r := byType[lt]
		match := got.MatchedFlagged == sc.ExpectFlagged
		switch sc.Category {
		case CategoryTruePositive:
			if match {
				r.tp++
			} else {
				r.fn++ // expected flagged, didn't get it
			}
		case CategoryTrueNegative:
			if match {
				r.tn++
			} else {
				r.fp++ // didn't expect flagged, got it
			}
		case CategoryFalsePositive:
			if match {
				r.tn++ // desired: not flagged
			} else {
				r.fp++
			}
		case CategoryFalseNegative:
			if match {
				r.tp++ // desired: flagged
			} else {
				r.fn++
			}
		}
	}

	types := make([]string, 0, len(byType))
	for k := range byType {
		types = append(types, k)
	}
	sort.Strings(types)

	var b strings.Builder
	b.WriteString("# Claim-type stress test matrix\n\n")
	b.WriteString("Combined precision/recall estimates from the stress corpus (`test/stress/`).\n")
	b.WriteString("Regenerate with `STRESS_WRITE_REPORTS=1 go test -tags stress -run TestWriteMatrix ./test/stress/...`.\n\n")
	b.WriteString("| Claim type | TP | TN | FP | FN | Precision (est.) | Recall (est.) | Priority |\n")
	b.WriteString("|----------|----|----|----|----|------------------|---------------|----------|\n")

	type priority struct {
		lt     string
		fp, fn int
		score  float64
	}
	var prios []priority

	for _, lt := range types {
		r := byType[lt]
		prec := estPrecision(r.tp, r.fp)
		rec := estRecall(r.tp, r.fn)
		pri := "low"
		gap := r.fp + r.fn
		if gap >= 8 {
			pri = "critical"
		} else if gap >= 4 {
			pri = "high"
		} else if gap >= 2 {
			pri = "medium"
		}
		prios = append(prios, priority{lt, r.fp, r.fn, float64(gap)})
		b.WriteString(fmt.Sprintf("| `%s` | %d | %d | %d | %d | %s | %s | %s |\n",
			lt, r.tp, r.tn, r.fp, r.fn, prec, rec, pri))
	}

	sort.Slice(prios, func(i, j int) bool {
		return prios[i].score > prios[j].score
	})

	b.WriteString("\n## Implementation priority (Wave 3)\n\n")
	b.WriteString("Top fixes for a follow-up implementation PR, ordered by total gap count (FP+FN):\n\n")
	for i, p := range prios {
		if p.fp+p.fn == 0 {
			continue
		}
		b.WriteString(fmt.Sprintf("%d. **`%s`** — %d FP, %d FN → see [stress-test-mitigations.md](stress-test-mitigations.md)\n", i+1, p.lt, p.fp, p.fn))
	}

	b.WriteString("\n## Per-family reports\n\n")
	for _, f := range []string{"file_prose", "stub_noaction", "git", "shell", "consistency", "live_session"} {
		b.WriteString(fmt.Sprintf("- [%s](../test/stress/reports/%s.md)\n", f, f))
	}

	out := filepath.Join("..", "..", "docs", "stress-test-matrix.md")
	if err := os.WriteFile(out, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func estPrecision(tp, fp int) string {
	denom := tp + fp
	if denom == 0 {
		return "—"
	}
	return fmt.Sprintf("%.0f%%", 100*float64(tp)/float64(denom))
}

func estRecall(tp, fn int) string {
	denom := tp + fn
	if denom == 0 {
		return "—"
	}
	return fmt.Sprintf("%.0f%%", 100*float64(tp)/float64(denom))
}
