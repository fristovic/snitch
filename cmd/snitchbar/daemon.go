//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/fristovic/snitch/internal/ipc"
	"github.com/fristovic/snitch/internal/platform"
)

type daemonMgr struct {
	mu    sync.Mutex
	cmd   *exec.Cmd
	path  string
	owned bool // true when this mgr started snitchd (not merely connected to an existing one)
}

func newDaemonMgr() *daemonMgr {
	return &daemonMgr{path: resolveSnitchdPath()}
}

func resolveSnitchdPath() string {
	if p := os.Getenv("SNITCHD_PATH"); p != "" {
		return p
	}
	if exe, err := os.Executable(); err == nil {
		bundled := filepath.Join(filepath.Dir(exe), "snitchd")
		if st, err := os.Stat(bundled); err == nil && !st.IsDir() {
			return bundled
		}
	}
	if p, err := exec.LookPath("snitchd"); err == nil {
		return p
	}
	return ""
}

func (m *daemonMgr) reachable(socket string) bool {
	client, err := ipc.Connect(socket)
	if err != nil {
		return false
	}
	_ = client.Close()
	return true
}

func (m *daemonMgr) ensureRunning(socket string) error {
	if m.reachable(socket) {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil && m.cmd.Process != nil {
		if m.reachable(socket) {
			return nil
		}
	}

	if m.path == "" {
		return fmt.Errorf("snitchd binary not found")
	}

	logPath, _ := platform.Resolve()
	var logFile *os.File
	if logPath != nil {
		_ = os.MkdirAll(logPath.DataDir, 0o700)
		logFile, _ = os.OpenFile(logPath.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	}

	cmd := exec.Command(m.path)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	m.cmd = cmd
	m.owned = true

	if !waitForSocket(socket, 10*time.Second) {
		_ = m.stopProcessLocked()
		return fmt.Errorf("snitchd did not become ready")
	}
	return nil
}

func (m *daemonMgr) stop(_ string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.owned {
		return
	}
	_ = m.stopProcessLocked()
}

func (m *daemonMgr) stopProcessLocked() error {
	if m.cmd == nil || m.cmd.Process == nil {
		return nil
	}
	_ = m.cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- m.cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = m.cmd.Process.Kill()
		<-done
	}
	m.cmd = nil
	m.owned = false
	return nil
}

func waitForSocket(socket string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if client, err := ipc.Connect(socket); err == nil {
			_ = client.Close()
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}
