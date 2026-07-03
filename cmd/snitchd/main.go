package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fristovic/snitch/internal/capture"
	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/ipc"
	"github.com/fristovic/snitch/internal/notify"
	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/report"
	"github.com/fristovic/snitch/internal/transcript"
	"github.com/fristovic/snitch/internal/verify"
	"github.com/fristovic/snitch/internal/version"
)

func main() {
	cfg, paths, err := config.LoadFromPlatform()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	store, err := record.Open(paths.DataDir)
	if err != nil {
		slog.Error("store open failed", "err", err)
		os.Exit(1)
	}
	defer store.Close()

	deviceID, err := store.EnsureDeviceID()
	if err != nil {
		slog.Error("device id failed", "err", err)
		os.Exit(1)
	}

	bus := event.NewBus()
	defer bus.Close()

	watcher := transcript.NewWatcher(bus, cfg.Cursor)
	if err := watcher.Start(); err != nil {
		slog.Error("cursor watcher failed", "err", err)
		os.Exit(1)
	}

	capEngine := capture.New(bus)
	capEngine.Start()

	ipcServer := ipc.NewServer(ipc.Deps{
		Store:      store,
		Config:     cfg,
		ConfigPath: paths.ConfigPath,
		Version:    version.Version,
	})

	verifyEngine := verify.NewEngine(bus, store, cfg.Verification, deviceID)
	verifyEngine.OnVerified(func(p event.RunVerifiedPayload) {
		ipcServer.Broadcast("run.completed", p)
		notify.MaybeNotify(store, p, cfg.Notifications)
	})
	verifyEngine.Start()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reporter := report.New(cfg.Analytics, store, version.Version)
	reporter.Start(ctx)

	startRetention(ctx, store, cfg.Retention)

	if err := ipcServer.Listen(); err != nil {
		slog.Error("ipc listen failed", "err", err)
		os.Exit(1)
	}
	defer ipcServer.Close()

	slog.Info("snitchd started", "version", version.Version, "socket", cfg.Daemon.SocketPath)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	slog.Info("shutting down")
	cancel()
	_ = watcher.Stop()
	capEngine.Stop()
	if err := store.Vacuum(); err != nil {
		slog.Warn("vacuum failed", "err", err)
	}
	time.Sleep(100 * time.Millisecond)
}

func startRetention(ctx context.Context, store *record.Store, cfg config.RetentionConfig) {
	if cfg.MaxDays <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := store.ApplyRetention(cfg.MaxDays, cfg.KeepFailures); err != nil {
					slog.Warn("retention failed", "err", err)
				}
			}
		}
	}()
}
