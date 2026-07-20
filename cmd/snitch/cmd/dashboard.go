package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fristovic/snitch/internal/claims"
	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/ipc"
	"github.com/fristovic/snitch/internal/platform"
	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/textutil"
	"github.com/spf13/cobra"
)

type viewMode int

const (
	modeRuns viewMode = iota
	modeFlagged
)

// dashboardHarness filters the dashboard to one harness when set via --harness.
var dashboardHarness string

var dashboardSelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("236"))

type filterState struct {
	Verdict    string // "", "snitched", "all"
	ClaimType  string
	Project    string
	Search     string
	ShowPasses bool
	Harness    string // "", or a harness name (cursor/claude/codex/pi/opencode)
}

type dashboardModel struct {
	client *ipc.Client
	cfg    config.TUIConfig
	status record.DaemonStatus
	runs   []record.Run
	flaggedClaims []record.ClaimWithRun
	claimsByRun map[string][]record.Claim
	filter filterState
	mode   viewMode
	cursor int
	width  int
	height int
	err    error
}

type tickMsg struct{}
type refreshMsg struct{}

func (m dashboardModel) Init() tea.Cmd {
	return tea.Batch(tickCmd(m.cfg.RefreshMS), m.loadData())
}

func tickCmd(ms int) tea.Cmd {
	if ms <= 0 {
		ms = 500
	}
	return tea.Tick(time.Duration(ms)*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m dashboardModel) loadData() tea.Cmd {
	return func() tea.Msg {
		return refreshMsg{}
	}
}

func (m *dashboardModel) refresh() error {
	stData, err := m.client.Call("status", nil)
	if err != nil {
		return err
	}
	_ = json.Unmarshal(stData, &m.status)

	if m.mode == modeFlagged {
		params := map[string]any{"false_claims_only": true, "limit": m.cfg.MaxRunsVisible}
		if m.filter.ClaimType != "" {
			params["claim_type"] = m.filter.ClaimType
		}
		if m.filter.Project != "" {
			params["project_path"] = m.filter.Project
		}
		if m.filter.Search != "" {
			params["search"] = m.filter.Search
		}
		data, err := m.client.Call("get_claims", params)
		if err != nil {
			return err
		}
		_ = json.Unmarshal(data, &m.flaggedClaims)
		if m.cursor >= len(m.flaggedClaims) {
			m.cursor = max(0, len(m.flaggedClaims)-1)
		}
		return nil
	}

	params := map[string]any{"limit": m.cfg.MaxRunsVisible}
	if m.filter.Verdict == "snitched" || !m.filter.ShowPasses {
		params["failures_only"] = true
	}
	if m.filter.Project != "" {
		params["project_path"] = m.filter.Project
	}
	if m.filter.Search != "" {
		params["search"] = m.filter.Search
	}
	if m.filter.Harness != "" {
		params["harness"] = m.filter.Harness
	}
	data, err := m.client.Call("get_runs", params)
	if err != nil {
		return err
	}
	_ = json.Unmarshal(data, &m.runs)
	if m.cursor >= len(m.runs) {
		m.cursor = max(0, len(m.runs)-1)
	}
	return nil
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tickMsg:
		return m, tea.Batch(tickCmd(m.cfg.RefreshMS), m.loadData())
	case refreshMsg:
		if err := m.refresh(); err != nil {
			m.err = err
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			maxIdx := m.listLen() - 1
			if m.cursor < maxIdx {
				m.cursor++
			}
		case "tab":
			if m.mode == modeRuns {
				m.mode = modeFlagged
			} else {
				m.mode = modeRuns
			}
			m.cursor = 0
			return m, m.loadData()
		case "v":
			m.filter = cycleVerdictFilter(m.filter)
			m.cursor = 0
			return m, m.loadData()
		case "t":
			m.filter = cycleClaimTypeFilter(m.filter)
			m.cursor = 0
			return m, m.loadData()
		case "/":
			m.filter.Search = promptSearch()
			m.cursor = 0
			return m, m.loadData()
		case "esc":
			m.filter.Search = ""
			return m, m.loadData()
		}
	}
	return m, nil
}

func (m dashboardModel) listLen() int {
	if m.mode == modeFlagged {
		return len(m.flaggedClaims)
	}
	return len(m.runs)
}

func (m dashboardModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	modeLabel := "runs"
	if m.mode == modeFlagged {
		modeLabel = "flagged"
	}
	header := headerStyle.Render(fmt.Sprintf("Snitch — %s", modeLabel))
	stats := fmt.Sprintf("runs=%d snitched=%d projects=%d sessions=%d",
		m.status.TotalRuns, m.status.SnitchedRuns, m.status.ProjectsWatched, m.status.SessionsSeen)
	filters := filterStyle.Render(fmt.Sprintf(
		"verdict=%s type=%s project=%s search=%q | tab mode  v verdict  t type  / search",
		displayVerdict(m.filter), orDash(m.filter.ClaimType), orDash(m.filter.Project), m.filter.Search,
	))

	width := m.width
	if width <= 0 {
		width = 80
	}
	height := m.height
	if height <= 0 {
		height = 24
	}

	listW, detailW, listRows, _, stack := layoutMetrics(width, height)

	var listBody, detailBody string
	if m.mode == modeFlagged {
		listBody, detailBody = m.viewFlaggedClaims(listW, detailW, listRows)
	} else {
		listBody, detailBody = m.viewRuns(listW, detailW, listRows)
	}

	listPane := lipgloss.NewStyle().Width(listW).Render(listBody)
	detailPane := lipgloss.NewStyle().Width(detailW).Render(detailBody)

	var body string
	if stack {
		body = listPane + "\n" + filterStyle.Render(strings.Repeat("─", min(width, 40))) + "\n" + detailPane
	} else {
		body = lipgloss.JoinHorizontal(lipgloss.Top, listPane, " │ ", detailPane)
	}
	help := filterStyle.Render("↑/↓ navigate  tab runs/flagged  q quit")
	return header + "\n" + stats + "\n" + filters + "\n" + body + "\n" + help
}

func layoutMetrics(width, height int) (listW, detailW, listRows, detailRows int, stack bool) {
	bodyRows := height - 6
	if bodyRows < 6 {
		bodyRows = 6
	}
	stack = width < 100
	if stack {
		listW = width
		detailW = width
		listRows = bodyRows / 2
		if listRows < 4 {
			listRows = 4
		}
		detailRows = bodyRows - listRows - 1
		if detailRows < 3 {
			detailRows = 3
		}
		return listW, detailW, listRows, detailRows, stack
	}
	listW = width/2 - 2
	if listW < 28 {
		listW = 28
	}
	detailW = width - listW - 3
	if detailW < 28 {
		detailW = 28
	}
	return listW, detailW, bodyRows, bodyRows, stack
}

func (m dashboardModel) viewRuns(listW, detailW, listRows int) (string, string) {
	var list strings.Builder
	if len(m.runs) == 0 {
		list.WriteString("  (no runs)\n")
	} else {
		start, end := visibleWindow(m.cursor, len(m.runs), listRows)
		for i := start; i < end; i++ {
			r := m.runs[i]
			summary := runListSummary(r, listW-18)
			line := fmt.Sprintf("  %s %-4s %s", shortID(r.ID), r.Verdict, summary)
			if i == m.cursor {
				line = dashboardSelStyle.Render(textutil.TruncateRunes("> "+shortID(r.ID)+" "+string(r.Verdict)+" "+summary, listW))
			} else {
				line = textutil.TruncateRunes(line, listW)
			}
			list.WriteString(line + "\n")
		}
	}

	detail := "(select a run)"
	if m.cursor < len(m.runs) {
		r := m.runs[m.cursor]
		prompt := textutil.OneLine(formatPrompt(r.Command), detailW*3)
		detail = fmt.Sprintf("Run %s\nVerdict: %s\nProject: %s\nSession: %s\nHarness: %s\nTool calls: %d  flagged: %d\n\nPrompt:\n%s",
			shortID(r.ID), r.Verdict, textutil.OneLine(r.ProjectPath, detailW), shortID(r.SessionID),
			orDash(r.Harness), r.ToolCallCount, r.FalseClaims, prompt)
		if m.client != nil {
			runClaims, _ := m.fetchClaims(r.ID)
			if len(runClaims) > 0 {
				detail += "\n\nClaims:"
				for _, c := range runClaims {
					if !record.ShowClaimInDetail(c, 2) {
						continue
					}
					detail += "\n\n" + claims.RichDetail(claims.FromRecord(c))
				}
			}
		}
	}
	return list.String(), detail
}

func (m dashboardModel) viewFlaggedClaims(listW, detailW, listRows int) (string, string) {
	var list strings.Builder
	if len(m.flaggedClaims) == 0 {
		list.WriteString("  (no flagged claims)\n")
	} else {
		start, end := visibleWindow(m.cursor, len(m.flaggedClaims), listRows)
		for i := start; i < end; i++ {
			c := m.flaggedClaims[i]
			summary := claims.ShortSummary(claims.FromRecord(c.Claim), listW-20)
			plain := fmt.Sprintf("  %s %s", c.RunCreated.Format("15:04"), summary)
			if i == m.cursor {
				list.WriteString(dashboardSelStyle.Render(textutil.TruncateRunes("> "+c.RunCreated.Format("15:04")+" "+summary, listW)) + "\n")
			} else {
				list.WriteString(textutil.TruncateRunes(plain, listW) + "\n")
			}
		}
	}

	detail := "(select a claim)"
	if m.cursor < len(m.flaggedClaims) {
		c := m.flaggedClaims[m.cursor]
		detail = fmt.Sprintf("Project: %s\nSession: %s\nRun: %s\n\n%s",
			textutil.OneLine(c.ProjectPath, detailW), shortID(c.SessionID), shortID(c.RunID),
			claims.RichDetail(claims.FromRecord(c.Claim)))
	}
	return list.String(), detail
}

// visibleWindow returns a [start,end) window of size rows centered on cursor.
func visibleWindow(cursor, total, rows int) (int, int) {
	if rows < 1 {
		rows = 1
	}
	if total <= rows {
		return 0, total
	}
	start := cursor - rows/2
	if start < 0 {
		start = 0
	}
	end := start + rows
	if end > total {
		end = total
		start = end - rows
		if start < 0 {
			start = 0
		}
	}
	return start, end
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func runListSummary(r record.Run, max int) string {
	if s := textutil.OneLine(formatPrompt(r.Command), max); s != "" {
		return s
	}
	if r.ProjectPath != "" {
		return textutil.OneLine(r.ProjectPath, max)
	}
	return ""
}

func (m *dashboardModel) fetchClaims(runID string) ([]record.Claim, error) {
	if m.claimsByRun == nil {
		m.claimsByRun = make(map[string][]record.Claim)
	}
	if c, ok := m.claimsByRun[runID]; ok {
		return c, nil
	}
	data, err := m.client.Call("get_run", map[string]string{"id": runID})
	if err != nil {
		return nil, err
	}
	var resp struct {
		Claims []record.Claim `json:"claims"`
	}
	_ = json.Unmarshal(data, &resp)
	m.claimsByRun[runID] = resp.Claims
	return resp.Claims, nil
}

func cycleVerdictFilter(f filterState) filterState {
	switch f.Verdict {
	case "":
		f.Verdict = "snitched"
		f.ShowPasses = false
	case "snitched":
		f.Verdict = "all"
		f.ShowPasses = true
	default:
		f.Verdict = ""
		f.ShowPasses = false
	}
	return f
}

func cycleClaimTypeFilter(f filterState) filterState {
	types := append([]string{""}, claims.AllFilterTypes()...)
	for i, t := range types {
		if f.ClaimType == t {
			f.ClaimType = types[(i+1)%len(types)]
			return f
		}
	}
	f.ClaimType = types[1]
	return f
}

func displayVerdict(f filterState) string {
	if f.Verdict == "all" {
		return "all"
	}
	return "snitched"
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func promptSearch() string {
	fmt.Print("search: ")
	var s string
	_, _ = fmt.Scanln(&s)
	return strings.TrimSpace(s)
}

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Interactive claim-verifier TUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		if os.Getenv("TERM") == "" {
			return fmt.Errorf("dashboard requires a terminal")
		}
		client, err := ipc.Connect(resolveSocket())
		if err != nil {
			daemonNotRunning()
			return nil
		}

		paths, _ := platform.Resolve()
		cfg, _ := config.Load(paths.ConfigPath)
		tuiCfg := cfg.Display.TUI
		if tuiCfg.MaxRunsVisible <= 0 {
			tuiCfg.MaxRunsVisible = 100
		}
		if tuiCfg.RefreshMS <= 0 {
			tuiCfg.RefreshMS = 500
		}

		m := dashboardModel{
			client: client,
			cfg:    tuiCfg,
			filter: filterState{Verdict: "snitched", Harness: dashboardHarness},
		}

		p := tea.NewProgram(m, tea.WithAltScreen())
		_, err = p.Run()
		client.Close()
		return err
	},
}

func init() {
	dashboardCmd.Flags().StringVar(&dashboardHarness, "harness", "", "Filter to one harness (cursor, claude, codex, pi, opencode)")
}
