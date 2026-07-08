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
	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/ipc"
	"github.com/fristovic/snitch/internal/platform"
	"github.com/fristovic/snitch/internal/record"
	"github.com/getlantern/systray"
)

type trayApp struct {
	socket string
	daemon *daemonMgr

	mu    sync.Mutex
	state MenuState

	refreshCh chan struct{}

	status        *systray.MenuItem
	toggleItem    *systray.MenuItem
	showItem      *systray.MenuItem
	thumbUpItem   *systray.MenuItem
	thumbDownItem *systray.MenuItem
	missedItem    *systray.MenuItem
	shareItem     *systray.MenuItem
	dashboardItem *systray.MenuItem
	prefsItem     *systray.MenuItem
	quitItem      *systray.MenuItem

	shareLabels  bool // persisted as telemetry.share_by_default
	consentShown bool // persisted as telemetry.consent_shown
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
	a.showItem = systray.AddMenuItem("Show Last Lie", "Open full verification log for the latest lie")
	a.thumbUpItem = systray.AddMenuItem("👍 Was Snitch right", "Mark the last verdict correct — helps train Snitch")
	a.thumbDownItem = systray.AddMenuItem("👎 Was Snitch wrong", "Mark the last verdict incorrect — helps train Snitch")
	a.missedItem = systray.AddMenuItem("Report Missed Lie…", "Report a lie Snitch missed (opens terminal)")
	a.shareItem = systray.AddMenuItemCheckbox("Share labels anonymously", "Share verdict metadata to train Snitch — no code or text leaves your machine", false)
	a.dashboardItem = systray.AddMenuItem("Open Dashboard…", "Browse runs and lies in the interactive TUI")
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
		case <-a.showItem.ClickedCh:
			a.acknowledgeAlert()
			a.loadLatestLie()
			a.showLastLie()
		case <-a.thumbUpItem.ClickedCh:
			a.submitLabel("correct")
		case <-a.thumbDownItem.ClickedCh:
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
		return
	}
	a.mu.Lock()
	a.state.Lie = &claims[0]
	a.mu.Unlock()
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
	var cfg struct {
		Telemetry struct {
			ShareByDefault bool `yaml:"share_by_default" json:"share_by_default"`
			ConsentShown   bool `yaml:"consent_shown" json:"consent_shown"`
		} `json:"telemetry"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		return
	}
	a.mu.Lock()
	a.shareLabels = cfg.Telemetry.ShareByDefault
	a.consentShown = cfg.Telemetry.ConsentShown
	a.mu.Unlock()
	if a.shareLabels {
		a.shareItem.Check()
	} else {
		a.shareItem.Uncheck()
	}
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
// detected lie. It never enables sharing — it only informs; the user opts in
// via the checkbox or CLI.
func (a *trayApp) maybeShowConsent() {
	a.mu.Lock()
	shown := a.consentShown
	a.consentShown = true
	a.mu.Unlock()
	if shown {
		return
	}
	a.setConfig("telemetry.consent_shown", "true")
	a.flashTitle("Snitch can train a smarter lie detector from anonymous verdicts — enable 'Share labels anonymously' to help. No code ever leaves your machine.")
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
	if st.Lie == nil {
		a.showItem.Disable()
	} else {
		a.showItem.Enable()
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
