package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/ipc"
	"github.com/fristovic/snitch/internal/platform"
	"github.com/fristovic/snitch/internal/record"
	"github.com/spf13/cobra"
)

type viewMode int

const (
	modeRuns viewMode = iota
	modeLies
)

type filterState struct {
	Verdict     string // "", "snitched", "all"
	ClaimType   string
	Project     string
	Search      string
	ShowPasses  bool
}

type dashboardModel struct {
	client      *ipc.Client
	cfg         config.TUIConfig
	status      record.DaemonStatus
	runs        []record.Run
	lies        []record.LieClaim
	claims      map[string][]record.Claim
	filter      filterState
	mode        viewMode
	cursor      int
	width       int
	height      int
	err         error
	watchCtx    context.Context
	watchCancel context.CancelFunc
}

type tickMsg struct{}
type refreshMsg struct{}
type eventMsg ipc.EventMsg

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

	if m.mode == modeLies {
		params := map[string]any{"lies_only": true, "limit": m.cfg.MaxRunsVisible}
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
		_ = json.Unmarshal(data, &m.lies)
		if m.cursor >= len(m.lies) {
			m.cursor = max(0, len(m.lies)-1)
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
	case eventMsg:
		var p event.RunVerifiedPayload
		if json.Unmarshal(msg.Data, &p) == nil {
			if m.filter.Verdict != "all" && p.Verdict == record.VerdictPass {
				return m, nil
			}
		}
		_ = m.refresh()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.watchCancel != nil {
				m.watchCancel()
			}
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
				m.mode = modeLies
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
		case "p":
			m.filter = cycleProjectFilter(m.filter, m.status)
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
	if m.mode == modeLies {
		return len(m.lies)
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
	if m.mode == modeLies {
		modeLabel = "lies"
	}
	header := headerStyle.Render(fmt.Sprintf("Snitch — %s", modeLabel))
	stats := fmt.Sprintf("runs=%d snitched=%d projects=%d sessions=%d",
		m.status.TotalRuns, m.status.SnitchedRuns, m.status.ProjectsWatched, m.status.SessionsSeen)
	filters := filterStyle.Render(fmt.Sprintf(
		"verdict=%s type=%s project=%s search=%q | tab mode v verdict t type p project / search",
		displayVerdict(m.filter), orDash(m.filter.ClaimType), orDash(m.filter.Project), m.filter.Search,
	))

	listW := m.width/2 - 2
	if listW < 20 {
		listW = 20
	}
	detailW := m.width - listW - 4
	if detailW < 20 {
		detailW = 20
	}

	var listBody, detailBody string
	if m.mode == modeLies {
		listBody, detailBody = m.viewLies(listW, detailW)
	} else {
		listBody, detailBody = m.viewRuns(listW, detailW)
	}

	listPane := lipgloss.NewStyle().Width(listW).Render(listBody)
	detailPane := lipgloss.NewStyle().Width(detailW).Render(detailBody)

	body := lipgloss.JoinHorizontal(lipgloss.Top, listPane, " │ ", detailPane)
	help := filterStyle.Render("\n↑/↓ navigate  tab runs/lies  q quit")
	return header + "\n" + stats + "\n" + filters + "\n\n" + body + help
}

func (m dashboardModel) viewRuns(listW, detailW int) (string, string) {
	var list strings.Builder
	for i, r := range m.runs {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%s %s", prefix, r.ID[:8], r.Verdict)
		if len(r.Command) > 0 {
			line += " " + truncateLog(r.Command, min(40, listW-20))
		}
		list.WriteString(truncateLog(line, listW) + "\n")
	}
	if len(m.runs) == 0 {
		list.WriteString("  (no runs)\n")
	}

	detail := "(select a run)\n"
	if m.cursor < len(m.runs) {
		r := m.runs[m.cursor]
		detail = fmt.Sprintf("Run %s\nVerdict: %s\nProject: %s\nSession: %s\nTool calls: %d\n\nPrompt:\n%s\n",
			r.ID, r.Verdict, r.ProjectPath, r.SessionID, r.ToolCallCount, truncateLog(r.Command, detailW))
		claims, _ := m.fetchClaims(r.ID)
		if len(claims) > 0 {
			detail += "\nClaims:\n"
			for _, c := range claims {
				if c.Severity < 2 && c.Verified > 0 {
					continue
				}
				detail += fmt.Sprintf("  [%s] %q → %s\n", c.ClaimType, truncateLog(c.Claimed, 50), c.Actual)
			}
		}
	}
	return list.String(), detail
}

func (m dashboardModel) viewLies(listW, detailW int) (string, string) {
	var list strings.Builder
	for i, c := range m.lies {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%s %s %q", prefix, c.ClaimType, c.RunCreated.Format("15:04"), truncateLog(c.Claimed, min(30, listW-25)))
		list.WriteString(truncateLog(line, listW) + "\n")
	}
	if len(m.lies) == 0 {
		list.WriteString("  (no lies)\n")
	}

	detail := "(select a lie)\n"
	if m.cursor < len(m.lies) {
		c := m.lies[m.cursor]
		detail = fmt.Sprintf("Lie: %s\nProject: %s\nSession: %s\nRun: %s\n\nClaimed:\n%s\n\nEvidence:\n%s\nVerifier: %s (sev %d)\n",
			c.ClaimType, c.ProjectPath, c.SessionID, c.RunID,
			c.Claimed, c.Actual, c.Verifier, c.Severity)
	}
	return list.String(), truncateLog(detail, detailW*4)
}

func (m *dashboardModel) fetchClaims(runID string) ([]record.Claim, error) {
	if m.claims == nil {
		m.claims = make(map[string][]record.Claim)
	}
	if c, ok := m.claims[runID]; ok {
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
	m.claims[runID] = resp.Claims
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
	types := []string{"", "test_pass", "committed", "pushed", "file_created", "file_modified", "no_action"}
	for i, t := range types {
		if f.ClaimType == t {
			f.ClaimType = types[(i+1)%len(types)]
			return f
		}
	}
	f.ClaimType = types[1]
	return f
}

func cycleProjectFilter(f filterState, st record.DaemonStatus) filterState {
	if f.Project != "" {
		f.Project = ""
		return f
	}
	if st.ProjectsWatched > 0 {
		f.Project = "(set via / search)"
	}
	return f
}

func displayVerdict(f filterState) string {
	if f.Verdict == "all" {
		return "all"
	}
	if f.Verdict == "snitched" || !f.ShowPasses {
		return "snitched"
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Interactive lie-detector TUI",
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

		ctx, cancel := context.WithCancel(context.Background())
		m := dashboardModel{
			client:      client,
			cfg:         tuiCfg,
			filter:      filterState{Verdict: "snitched"},
			watchCtx:    ctx,
			watchCancel: cancel,
		}

		go func() {
			_ = ipc.Watch(ctx, resolveSocket(), func(msg ipc.EventMsg) error {
				return nil
			})
		}()

		p := tea.NewProgram(m, tea.WithAltScreen())
		_, err = p.Run()
		cancel()
		client.Close()
		return err
	},
}
