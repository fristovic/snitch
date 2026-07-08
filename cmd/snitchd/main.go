package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fristovic/snitch/internal/capture"
	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/harness"
	"github.com/fristovic/snitch/internal/ipc"
	"github.com/fristovic/snitch/internal/notify"
	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/report"
	"github.com/fristovic/snitch/internal/verify"
	"github.com/fristovic/snitch/internal/version"
)

func main() {
	cfg, paths, err := config.LoadFromPlatform()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}
	applyLogLevel(cfg.Daemon.LogLevel)

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

	registry := harness.NewRegistry()
	stoppers, err := harness.StartIngestion(bus, cfg, registry)
	if err != nil {
		slog.Error("ingestion failed", "err", err)
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

	verifyEngine := verify.NewEngine(bus, store, cfg.Verification, deviceID, registry.ShellResolver)
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
	startTelemetrySync(ctx, store, cfg.Telemetry, deviceID, version.Version, enabledPlatforms(cfg))

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
	for _, s := range stoppers {
		_ = s.Stop()
	}
	capEngine.Stop()
	if err := store.Vacuum(); err != nil {
		slog.Warn("vacuum failed", "err", err)
	}
	time.Sleep(100 * time.Millisecond)
}

// enabledPlatforms returns the names of harnesses enabled in config.
func enabledPlatforms(cfg *config.Config) []string {
	var out []string
	for _, name := range config.HarnessNames() {
		if pc, ok := cfg.Platforms.ForHarness(name); ok && pc.Enabled {
			out = append(out, name)
		}
	}
	return out
}

// applyLogLevel configures the global slog level from daemon.log_level.
func applyLogLevel(level string) {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	slog.SetLogLoggerLevel(l)
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

// startTelemetrySync forwards labeled-and-shared verdicts to the training
// pipeline. Opt-in only: no-ops entirely when telemetry.enabled is false.
// Labels are metadata-only (claim type, harness, verdict, label) — no code,
// file paths, or claim text ever leaves the machine. Offline POSTs are retried
// on the next tick (labels stay label_synced=0 until delivered).
func startTelemetrySync(ctx context.Context, store *record.Store, cfg config.TelemetryConfig, deviceID, version string, enabledPlatforms []string) {
	if !cfg.Enabled {
		return
	}
	interval := time.Duration(cfg.IntervalM) * time.Minute
	if interval <= 0 {
		interval = 60 * time.Minute
	}
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://telemetry.snitchworks.com/api/v1/telemetry/labels"
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// One-time device registration per daemon start (best effort).
		registerDevice(endpoint, deviceID, version, enabledPlatforms)
		syncOnce := func() {
			labels, err := store.UnsyncedLabels(100)
			if err != nil {
				return
			}
			missed, err := store.UnsyncedMissedClaims(100)
			if err == nil {
				labels = append(labels, missed...)
			}
			if len(labels) == 0 {
				return
			}
			if err := postLabels(endpoint, deviceID, version, labels); err != nil {
				slog.Debug("telemetry sync failed (will retry)", "err", err)
				return // leave unsynced; retry next tick
			}
			var runIDs []string
			var missedIDs []int64
			for _, l := range labels {
				if l.MissedID != 0 {
					missedIDs = append(missedIDs, l.MissedID)
				} else {
					runIDs = append(runIDs, l.RunID)
				}
			}
			if err := store.MarkLabelsSynced(runIDs); err != nil {
				slog.Warn("mark labels synced failed", "err", err)
			}
			if err := store.MarkMissedClaimsSynced(missedIDs); err != nil {
				slog.Warn("mark missed claims synced failed", "err", err)
			}
			slog.Info("telemetry sync forwarded", "count", len(labels))
		}
		// Drain once shortly after startup, then on the interval.
		time.Sleep(10 * time.Second)
		syncOnce()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				syncOnce()
			}
		}
	}()
}

// postLabels sends a batch of labeled verdicts to the telemetry endpoint.
// The payload is metadata-only; no claim text, code, or file paths.
func postLabels(endpoint, deviceID, version string, labels []record.RunLabel) error {
	type sharePayload struct {
		DeviceID string            `json:"device_id"`
		Version  string            `json:"snitch_version"`
		Labels   []record.RunLabel `json:"labels"`
	}
	body, _ := json.Marshal(sharePayload{DeviceID: deviceID, Version: version, Labels: labels})
	return postJSON(endpoint, body)
}

// registerDevice announces this device to the telemetry pipeline (opt-in only;
// called solely from the telemetry sync goroutine). Metadata only: hashed
// device id, version, enabled platform names. Failure is silent — registration
// is best-effort and retried on the next daemon start.
func registerDevice(labelsEndpoint, deviceID, version string, platforms []string) {
	endpoint := strings.TrimSuffix(labelsEndpoint, "/labels") + "/register"
	body, _ := json.Marshal(map[string]any{
		"device_id":      deviceID,
		"snitch_version": version,
		"platforms":      platforms,
	})
	if err := postJSON(endpoint, body); err != nil {
		slog.Debug("telemetry register failed", "err", err)
	}
}

func postJSON(endpoint string, body []byte) error {
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("telemetry endpoint returned %d", resp.StatusCode)
	}
	return nil
}
