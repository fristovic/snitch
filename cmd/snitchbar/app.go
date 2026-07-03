//go:build darwin

package main

import (
	"context"
	"encoding/json"
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

	status     *systray.MenuItem
	toggleItem *systray.MenuItem
	copyItem   *systray.MenuItem
	browseItem *systray.MenuItem
	prefsItem  *systray.MenuItem
	quitItem   *systray.MenuItem
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
	a.copyItem = systray.AddMenuItem("Copy Last Lie", "")
	a.browseItem = systray.AddMenuItem("Browse Lies…", "")
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
		case <-a.copyItem.ClickedCh:
			a.acknowledgeAlert()
			a.loadLatestLie()
			a.copyLastLie()
		case <-a.browseItem.ClickedCh:
			a.acknowledgeAlert()
			_ = openTerminal("snitch lies")
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

func (a *trayApp) copyLastLie() {
	a.mu.Lock()
	lie := a.state.Lie
	a.mu.Unlock()
	if lie == nil {
		return
	}
	_ = copyToClipboard(FormatLieCopy(*lie))
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
}

func (a *trayApp) acknowledgeAlert() {
	a.mu.Lock()
	if a.state.Alert {
		a.state.Alert = false
	}
	a.mu.Unlock()
	a.signalRefresh()
}
