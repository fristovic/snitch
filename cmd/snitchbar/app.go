//go:build darwin

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fristovic/snitch/assets/menubar"
	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/ipc"
	"github.com/fristovic/snitch/internal/notify"
	"github.com/fristovic/snitch/internal/platform"
	"github.com/fristovic/snitch/internal/record"
	"github.com/getlantern/systray"
)

// flywheelUIEnabled gates Mark Correct/Incorrect, Report Missed Lie, and Share
// labels in the menu. Flip to true when shipping the labeling flywheel
// (Mark items still also require telemetry.enabled).
const flywheelUIEnabled = false

type trayApp struct {
	socket string
	daemon *daemonMgr

	mu    sync.Mutex
	state MenuState

	refreshCh chan struct{}

	status         *systray.MenuItem
	toggleItem     *systray.MenuItem
	previewItem    *systray.MenuItem // disabled context for latest lie
	viewItem       *systray.MenuItem
	markCorrect    *systray.MenuItem
	markIncorrect  *systray.MenuItem
	historyItem    *systray.MenuItem
	dashboardItem  *systray.MenuItem
	missedItem     *systray.MenuItem
	shareItem      *systray.MenuItem
	prefsItem      *systray.MenuItem
	quitItem       *systray.MenuItem

	shareLabels       bool // persisted as telemetry.share_by_default
	consentShown      bool // persisted as telemetry.consent_shown
	telemetryEnabled  bool // persisted as telemetry.enabled — gates label UI
	notifyCfg         config.NotificationsConfig
}

func newTrayApp(socket string) *trayApp {
	return &trayApp{
		socket:    socket,
		daemon:    newDaemonMgr(),
		refreshCh: make(chan struct{}, 1),
	}
}

func (a *trayApp) run() {
	systray.Run(a.onReady, a.onExit)
}

func (a *trayApp) onReady() {
	systray.SetTooltip("Snitch")
	a.status = systray.AddMenuItem("Starting…", "")
	a.status.Disable()
	systray.AddSeparator()
	a.toggleItem = systray.AddMenuItem("Stop Snitching", "Start or stop lie detection")
	systray.AddSeparator()
	a.previewItem = systray.AddMenuItem("No lies yet", "Most recent lie Snitch caught")
	a.previewItem.Disable()
	a.viewItem = systray.AddMenuItem("View Details…", "Open full verification log for the latest lie")
	a.markCorrect = systray.AddMenuItem("Mark Correct", "Snitch was right — this really was a lie (stored locally)")
	a.markIncorrect = systray.AddMenuItem("Mark Incorrect", "Snitch was wrong — false positive (stored locally)")
	systray.AddSeparator()
	a.historyItem = systray.AddMenuItem("History", "Browse history and training options")
	a.dashboardItem = a.historyItem.AddSubMenuItem("Open Dashboard…", "Browse runs and lies in the interactive TUI")
	a.missedItem = a.historyItem.AddSubMenuItem("Report Missed Lie…", "Report a lie Snitch missed (opens terminal)")
	a.shareItem = a.historyItem.AddSubMenuItemCheckbox("Share labels anonymously", "Share claim sentence + short context + claimed→actual to train Snitch — never prompts, code, or paths", false)
	a.markCorrect.Hide()
	a.markIncorrect.Hide()
	a.missedItem.Hide()
	a.shareItem.Hide()
	systray.AddSeparator()
	a.prefsItem = systray.AddMenuItem("Preferences…", "")
	a.quitItem = systray.AddMenuItem("Quit Snitch Bar", "")

	go a.handleClicks()
	go a.pollLoop()
	go a.watchLoop()
	go a.refreshLoop()

	go a.startWatching()
}

func (a *trayApp) onExit() {
	a.daemon.stop(a.socket)
}

func (a *trayApp) handleClicks() {
	for {
		select {
		case <-a.viewItem.ClickedCh:
			a.acknowledgeAlert()
			a.loadLatestLie()
			a.showLastLie()
		case <-a.markCorrect.ClickedCh:
			a.submitLabel("correct")
		case <-a.markIncorrect.ClickedCh:
			a.submitLabel("incorrect")
		case <-a.missedItem.ClickedCh:
			a.acknowledgeAlert()
			_ = openTerminal(`snitch label missed --claimed "what the agent said" --actual "what actually happened"`)
		case <-a.shareItem.ClickedCh:
			a.toggleShare()
		case <-a.dashboardItem.ClickedCh:
			a.acknowledgeAlert()
			_ = openTerminal("snitch dashboard")
		case <-a.toggleItem.ClickedCh:
			a.mu.Lock()
			active := a.state.Connected && !a.state.Paused && !a.state.Starting
			a.mu.Unlock()
			if active {
				a.pauseWatching()
			} else {
				go a.startWatching()
			}
		case <-a.prefsItem.ClickedCh:
			a.acknowledgeAlert()
			if paths, err := platform.Resolve(); err == nil {
				_ = openPath(paths.ConfigPath)
			}
		case <-a.quitItem.ClickedCh:
			a.daemon.stop(a.socket)
			systray.Quit()
			os.Exit(0)
		}
	}
}

func (a *trayApp) startWatching() {
	a.mu.Lock()
	a.state.Paused = false
	a.state.Starting = true
	a.mu.Unlock()
	a.signalRefresh()

	if err := a.daemon.ensureRunning(a.socket); err != nil {
		a.mu.Lock()
		a.state.Starting = false
		a.state.Connected = false
		a.mu.Unlock()
		a.signalRefresh()
		return
	}

	a.mu.Lock()
	a.state.Starting = false
	a.state.Connected = true
	a.mu.Unlock()
	a.loadLatestLie()
	a.loadShareState()
	a.signalRefresh()
}

func (a *trayApp) pauseWatching() {
	a.daemon.stop(a.socket)
	a.mu.Lock()
	a.state.Paused = true
	a.state.Starting = false
	a.state.Connected = false
	a.state.Alert = false
	a.mu.Unlock()
	a.signalRefresh()
}

func (a *trayApp) showLastLie() {
	a.mu.Lock()
	lie := a.state.Lie
	a.mu.Unlock()
	if lie == nil {
		return
	}
	_ = openTerminal(fmt.Sprintf("snitch log --run %s", lie.RunID))
}

func (a *trayApp) pollLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		a.refreshConnection()
		<-ticker.C
	}
}

func (a *trayApp) watchLoop() {
	for {
		a.mu.Lock()
		paused := a.state.Paused
		a.mu.Unlock()
		if paused {
			time.Sleep(time.Second)
			continue
		}

		ctx, cancel := context.WithCancel(context.Background())
		err := ipc.Watch(ctx, a.socket, func(msg ipc.EventMsg) error {
			if msg.Event != "run.completed" {
				return nil
			}
			var p event.RunVerifiedPayload
			if err := json.Unmarshal(msg.Data, &p); err != nil {
				return nil
			}
			a.maybeNotify(p)
			if p.Verdict != record.VerdictFail {
				return nil
			}
			a.mu.Lock()
			a.state.Connected = true
			a.state.Alert = true
			a.mu.Unlock()
			a.loadLatestLie()
			a.maybeShowConsent()
			a.signalRefresh()
			return nil
		})
		cancel()

		a.mu.Lock()
		paused = a.state.Paused
		a.mu.Unlock()
		if paused {
			continue
		}

		if err != nil {
			a.mu.Lock()
			a.state.Connected = false
			a.state.Alert = false
			a.mu.Unlock()
			a.signalRefresh()
			go a.startWatching()
		}
		time.Sleep(3 * time.Second)
	}
}

func (a *trayApp) refreshLoop() {
	for range a.refreshCh {
		a.render()
	}
}

func (a *trayApp) signalRefresh() {
	select {
	case a.refreshCh <- struct{}{}:
	default:
	}
}

func (a *trayApp) refreshConnection() {
	a.mu.Lock()
	if a.state.Paused {
		a.mu.Unlock()
		return
	}
	a.mu.Unlock()

	client, err := ipc.Connect(a.socket)
	if err != nil {
		a.mu.Lock()
		a.state.Connected = false
		a.mu.Unlock()
		a.signalRefresh()
		return
	}
	defer client.Close()
	if _, err := client.Call("status", nil); err != nil {
		a.mu.Lock()
		a.state.Connected = false
		a.mu.Unlock()
		a.signalRefresh()
		return
	}
	a.mu.Lock()
	wasOffline := !a.state.Connected
	a.state.Connected = true
	a.mu.Unlock()
	if wasOffline {
		a.loadLatestLie()
	}
	a.signalRefresh()
}

func (a *trayApp) loadLatestLie() {
	client, err := ipc.Connect(a.socket)
	if err != nil {
		return
	}
	defer client.Close()
	data, err := client.Call("get_claims", map[string]any{"lies_only": true, "limit": 1})
	if err != nil {
		return
	}
	var claims []record.LieClaim
	if err := json.Unmarshal(data, &claims); err != nil || len(claims) == 0 {
		a.mu.Lock()
		a.state.Lie = nil
		a.mu.Unlock()
		a.signalRefresh()
		return
	}
	a.mu.Lock()
	a.state.Lie = &claims[0]
	a.mu.Unlock()
	a.signalRefresh()
}

// maybeNotify posts a Notification Center alert from Snitch Bar so macOS
// attributes it to the app bundle (Snitch icon) instead of Script Editor.
func (a *trayApp) maybeNotify(p event.RunVerifiedPayload) {
	a.mu.Lock()
	cfg := a.notifyCfg
	a.mu.Unlock()
	notify.Deliver(p, cfg)
}

// submitLabel records a "Was this right?" verdict for the latest lie. The
// shared flag comes explicitly from the persisted "Share labels anonymously"
// checkbox state.
func (a *trayApp) submitLabel(verdict string) {
	a.mu.Lock()
	lie := a.state.Lie
	shared := a.shareLabels
	a.mu.Unlock()
	if lie == nil {
		a.flashTitle("No lie to label yet")
		return
	}
	client, err := ipc.Connect(a.socket)
	if err != nil {
		return
	}
	defer client.Close()
	if _, err := client.Call("set_label", map[string]any{
		"run_id": lie.RunID,
		"label":  verdict,
		"shared": shared,
	}); err != nil {
		a.flashTitle("Couldn't save label")
		return
	}
	a.flashTitle("Thanks — this helps train Snitch")
}

// loadShareState reads the persisted share/consent settings from the daemon.
func (a *trayApp) loadShareState() {
	client, err := ipc.Connect(a.socket)
	if err != nil {
		return
	}
	defer client.Close()
	data, err := client.Call("get_config", nil)
	if err != nil {
		return
	}
	var cfg config.Config
	if json.Unmarshal(data, &cfg) != nil {
		return
	}
	a.mu.Lock()
	a.shareLabels = cfg.Telemetry.ShareByDefault
	a.consentShown = cfg.Telemetry.ConsentShown
	a.telemetryEnabled = cfg.Telemetry.Enabled
	a.notifyCfg = cfg.Notifications
	a.mu.Unlock()
	if a.shareLabels {
		a.shareItem.Check()
	} else {
		a.shareItem.Uncheck()
	}
	a.signalRefresh()
}

// toggleShare flips the anonymous-sharing opt-in and persists it via config.
func (a *trayApp) toggleShare() {
	a.mu.Lock()
	a.shareLabels = !a.shareLabels
	on := a.shareLabels
	a.mu.Unlock()
	if on {
		a.shareItem.Check()
	} else {
		a.shareItem.Uncheck()
	}
	a.setConfig("telemetry.share_by_default", fmt.Sprintf("%v", on))
}

// maybeShowConsent shows the one-time telemetry consent prompt on the first
// detected lie when the flywheel UI and telemetry sync are enabled. It never
// enables sharing — it only informs; the user opts in via the checkbox or CLI.
func (a *trayApp) maybeShowConsent() {
	if !flywheelUIEnabled {
		return
	}
	a.mu.Lock()
	shown := a.consentShown
	telemetryOn := a.telemetryEnabled
	a.consentShown = true
	a.mu.Unlock()
	if shown || !telemetryOn {
		return
	}
	a.setConfig("telemetry.consent_shown", "true")
	a.flashTitle("Share labels anonymously sends the claim sentence, short surrounding text, and claimed→actual — never prompts, code, or paths. Enable the checkbox to help train Snitch.")
}

// setConfig persists a config key through the daemon.
func (a *trayApp) setConfig(key, value string) {
	client, err := ipc.Connect(a.socket)
	if err != nil {
		return
	}
	defer client.Close()
	_, _ = client.Call("set_config", map[string]any{"key": key, "value": value})
}

// flashTitle briefly sets the menu-bar tooltip to msg, used for label feedback.
func (a *trayApp) flashTitle(msg string) {
	go func() {
		systray.SetTooltip(msg)
		time.Sleep(2500 * time.Millisecond)
		systray.SetTooltip("Snitch")
	}()
}

func setTrayIcon(icon, icon2x []byte) {
	if len(icon2x) > 0 {
		systray.SetTemplateIcon(icon2x, icon2x)
		return
	}
	systray.SetTemplateIcon(icon, icon)
}

func (a *trayApp) render() {
	a.mu.Lock()
	st := a.state
	telemetryOn := a.telemetryEnabled
	a.mu.Unlock()

	switch {
	case st.Paused:
		setTrayIcon(menubar.IconOffline, menubar.IconOffline2x)
	case !st.Connected:
		setTrayIcon(menubar.IconOffline, menubar.IconOffline2x)
	case st.Alert:
		setTrayIcon(menubar.IconAlert, menubar.IconAlert2x)
	default:
		setTrayIcon(menubar.IconIdle, menubar.IconIdle2x)
	}

	a.status.SetTitle(StatusLabel(st))
	a.toggleItem.SetTitle(ToggleLabel(st))
	if st.Starting {
		a.toggleItem.Disable()
	} else {
		a.toggleItem.Enable()
	}

	a.previewItem.SetTitle(LatestLiePreview(st.Lie))
	a.previewItem.Disable()
	if st.Lie == nil {
		a.viewItem.Disable()
	} else {
		a.viewItem.Enable()
	}

	if !flywheelUIEnabled {
		a.markCorrect.Hide()
		a.markIncorrect.Hide()
		a.missedItem.Hide()
		a.shareItem.Hide()
		return
	}
	a.missedItem.Show()
	a.shareItem.Show()
	// Mark items also require telemetry.enabled so we don't offer labeling
	// when sync cannot run.
	if !telemetryOn {
		a.markCorrect.Hide()
		a.markIncorrect.Hide()
		return
	}
	a.markCorrect.Show()
	a.markIncorrect.Show()
	if st.Lie == nil {
		a.markCorrect.Disable()
		a.markIncorrect.Disable()
	} else {
		a.markCorrect.Enable()
		a.markIncorrect.Enable()
	}
}

func (a *trayApp) acknowledgeAlert() {
	a.mu.Lock()
	if a.state.Alert {
		a.state.Alert = false
	}
	a.mu.Unlock()
	a.signalRefresh()
}
