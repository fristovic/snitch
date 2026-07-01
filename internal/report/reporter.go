package report

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/platform"
	"github.com/fristovic/snitch/internal/record"
)

// Reporter sends opt-in analytics.
type Reporter struct {
	cfg     config.AnalyticsConfig
	store   *record.Store
	version string
	priv    ed25519.PrivateKey
}

// New creates an analytics reporter.
func New(cfg config.AnalyticsConfig, store *record.Store, version string) *Reporter {
	return &Reporter{cfg: cfg, store: store, version: version}
}

// Start runs the periodic reporter.
func (r *Reporter) Start(ctx context.Context) {
	if !r.cfg.Enabled {
		return
	}
	r.loadOrCreateKey()
	interval := time.Duration(r.cfg.IntervalH) * time.Hour
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.report()
			}
		}
	}()
}

func (r *Reporter) loadOrCreateKey() {
	path := r.cfg.SigningKeyPath
	if path == "" {
		path = filepath.Join(platform.ExpandHome("~/.snitch"), "analytics_key")
	} else {
		path = platform.ExpandHome(path)
	}
	data, err := os.ReadFile(path)
	if err == nil && len(data) == ed25519.PrivateKeySize {
		r.priv = ed25519.PrivateKey(data)
		return
	}
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		slog.Warn("analytics key generation failed", "err", err)
		return
	}
	r.priv = priv
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	_ = os.WriteFile(path, priv, 0o600)
}

// DryRun builds payload without sending.
func (r *Reporter) DryRun() ([]byte, error) {
	return r.buildPayload()
}

func (r *Reporter) report() {
	payload, err := r.buildPayload()
	if err != nil {
		return
	}
	sig := ed25519.Sign(r.priv, payload)
	req, err := http.NewRequest(http.MethodPost, r.cfg.Endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Snitch-Signature", base64.StdEncoding.EncodeToString(sig))
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("analytics send failed", "err", err)
		return
	}
	resp.Body.Close()
}

func (r *Reporter) buildPayload() ([]byte, error) {
	deviceID, err := r.store.EnsureDeviceID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	start := now.Add(-24 * time.Hour)
	total, sevDist, err := r.store.AnalyticsStats(start)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"device_id":      deviceID,
		"snitch_version": r.version,
		"period_start":   start.Format(time.RFC3339),
		"period_end":     now.Format(time.RFC3339),
		"stats": map[string]any{
			"total_runs":            total,
			"models":                []any{},
			"severity_distribution": sevDist,
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if err := r.store.EnqueueAnalytics(start.Format(time.RFC3339), now.Format(time.RFC3339), data); err != nil {
		slog.Warn("analytics enqueue failed", "err", err)
	}
	return data, nil
}
