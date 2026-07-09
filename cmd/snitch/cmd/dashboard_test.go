package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/textutil"
)

func TestCycleVerdictFilter(t *testing.T) {
	f := filterState{}
	f = cycleVerdictFilter(f)
	if f.Verdict != "snitched" {
		t.Fatalf("expected snitched, got %s", f.Verdict)
	}
	f = cycleVerdictFilter(f)
	if f.Verdict != "all" || !f.ShowPasses {
		t.Fatalf("expected all, got %+v", f)
	}
}

func TestCycleClaimTypeFilter(t *testing.T) {
	f := filterState{}
	f = cycleClaimTypeFilter(f)
	if f.ClaimType != "test_pass" {
		t.Fatalf("expected test_pass, got %s", f.ClaimType)
	}
}

func TestOneLineCollapsesNewlines(t *testing.T) {
	in := "<timestamp>Thursday</timestamp>\n<user_query>\nhello\nworld"
	got := textutil.OneLine(formatPrompt(in), 80)
	if strings.Contains(got, "\n") {
		t.Fatalf("still has newline: %q", got)
	}
	if !strings.Contains(got, "hello") || !strings.Contains(got, "world") {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestVisibleWindow(t *testing.T) {
	start, end := visibleWindow(0, 100, 10)
	if start != 0 || end != 10 {
		t.Fatalf("got %d-%d", start, end)
	}
	start, end = visibleWindow(50, 100, 10)
	if end-start != 10 || start > 50 || end <= 50 {
		t.Fatalf("got %d-%d", start, end)
	}
}

func TestViewRunsSingleLineRows(t *testing.T) {
	m := dashboardModel{
		width:  80,
		height: 24,
		cursor: 0,
		runs: []record.Run{{
			ID:        "abcdef12-xxxx",
			Verdict:   record.VerdictFail,
			Command:   "<timestamp>Thursday, Jul 9</timestamp>\n<user_query>\nline1\nline2\nline3",
			CreatedAt: time.Now(),
		}},
	}
	list, _ := m.viewRuns(40, 40, 10)
	for i, line := range strings.Split(strings.TrimRight(list, "\n"), "\n") {
		if strings.Count(line, "\n") > 0 {
			t.Fatalf("row %d has embedded newline", i)
		}
	}
	if !strings.Contains(list, ">") {
		t.Fatalf("missing selection marker: %q", list)
	}
}

func TestDashboardViewNoLeadingGap(t *testing.T) {
	m := dashboardModel{
		width:  80,
		height: 24,
		cursor: 0,
		mode:   modeRuns,
		status: record.DaemonStatus{TotalRuns: 10, SnitchedRuns: 5, ProjectsWatched: 2, SessionsSeen: 3},
		filter: filterState{Verdict: "snitched"},
		runs: []record.Run{
			{ID: "abcdef12-1", Verdict: record.VerdictFail, Command: "<timestamp>Thu</timestamp>\n<user_query>\nhello\nworld"},
			{ID: "bbbbbbbb-2", Verdict: record.VerdictWarn, Command: "short prompt", CreatedAt: time.Now()},
		},
	}
	out := m.View()
	if !strings.HasPrefix(out, "Snitch") && !strings.Contains(out, "Snitch") {
		t.Fatalf("missing header: %q", out[:min(40, len(out))])
	}
	if strings.Contains(out, "\n\n\n") {
		t.Fatal("unexpected large blank gap")
	}
	if !strings.Contains(out, ">") {
		t.Fatal("missing selection marker")
	}
	// Multi-line prompts must not leak as separate list rows.
	if strings.Contains(out, "\nworld\n") {
		t.Fatalf("prompt newline leaked into view: %q", out)
	}
}
