package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fristovic/snitch/internal/platform"
	"gopkg.in/yaml.v3"
)

// Config is the root Snitch configuration.
type Config struct {
	Daemon        DaemonConfig        `yaml:"daemon" json:"daemon"`
	Platforms     PlatformsConfig     `yaml:"platforms" json:"platforms"`
	Verification  VerificationConfig  `yaml:"verification" json:"verification"`
	Analytics     AnalyticsConfig     `yaml:"analytics" json:"analytics"`
	Retention     RetentionConfig     `yaml:"retention" json:"retention"`
	Display       DisplayConfig       `yaml:"display" json:"display"`
	Notifications NotificationsConfig `yaml:"notifications" json:"notifications"`
	Telemetry     TelemetryConfig     `yaml:"telemetry" json:"telemetry"`
}

type DaemonConfig struct {
	DataDir    string `yaml:"data_dir" json:"data_dir"`
	SocketPath string `yaml:"socket_path" json:"socket_path"`
	LogLevel   string `yaml:"log_level" json:"log_level"`
}

// PlatformConfig configures one harness's ingestion. TranscriptWatchPath is
// the watch root for JSONL harnesses (Cursor, Claude, Codex, Pi); for OpenCode
// it is the path to opencode.db.
type PlatformConfig struct {
	Enabled             bool   `yaml:"enabled" json:"enabled"`
	TranscriptWatchPath string `yaml:"transcript_watch_path" json:"transcript_watch_path"`
}

// PlatformsConfig holds per-harness ingestion settings.
type PlatformsConfig struct {
	Cursor   PlatformConfig `yaml:"cursor" json:"cursor"`
	Claude   PlatformConfig `yaml:"claude" json:"claude"`
	Codex    PlatformConfig `yaml:"codex" json:"codex"`
	Pi       PlatformConfig `yaml:"pi" json:"pi"`
	OpenCode PlatformConfig `yaml:"opencode" json:"opencode"`
}

// byHarness is the single source of truth mapping harness names to platform
// config fields. ForHarness, Set, Get, and path expansion all iterate it.
func (p *PlatformsConfig) byHarness() map[string]*PlatformConfig {
	return map[string]*PlatformConfig{
		"cursor":   &p.Cursor,
		"claude":   &p.Claude,
		"codex":    &p.Codex,
		"pi":       &p.Pi,
		"opencode": &p.OpenCode,
	}
}

// HarnessNames returns every known harness name in stable order. This is the
// canonical list — CLI surfaces and the daemon iterate it instead of
// hardcoding their own copies.
func HarnessNames() []string {
	return []string{"cursor", "claude", "codex", "pi", "opencode"}
}

// ForHarness returns the PlatformConfig for a harness name.
func (p *PlatformsConfig) ForHarness(name string) (PlatformConfig, bool) {
	pc, ok := p.byHarness()[name]
	if !ok {
		return PlatformConfig{}, false
	}
	return *pc, true
}

// TelemetryConfig controls opt-in sharing of labeled verdicts to train a
// future semantic verifier. Off by default; users must explicitly enable.
type TelemetryConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	Endpoint       string `yaml:"endpoint" json:"endpoint"`
	IntervalM      int    `yaml:"interval_m" json:"interval_m"`
	ShareByDefault bool   `yaml:"share_by_default" json:"share_by_default"`
	// ConsentShown tracks whether the one-time telemetry consent prompt has
	// been displayed (Snitch Bar shows it on the first flagged claim).
	ConsentShown bool `yaml:"consent_shown" json:"consent_shown"`
}

type VerificationConfig struct {
	MaxConcurrentVerifications int                 `yaml:"max_concurrent_verifications" json:"max_concurrent_verifications"`
	ShellVerifier              ShellVerifierConfig `yaml:"shell_verifier" json:"shell_verifier"`
}

type ShellVerifierConfig struct {
	AllowRerun map[string]bool `yaml:"allow_rerun" json:"allow_rerun"`
}

type AnalyticsConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	Endpoint       string `yaml:"endpoint" json:"endpoint"`
	IntervalH      int    `yaml:"interval_h" json:"interval_h"`
	SigningKeyPath string `yaml:"signing_key_path" json:"signing_key_path"`
}

type RetentionConfig struct {
	MaxDays      int  `yaml:"max_days" json:"max_days"`
	KeepFailures bool `yaml:"keep_failures" json:"keep_failures"`
}

type DisplayConfig struct {
	TUI TUIConfig `yaml:"tui" json:"tui"`
}

type TUIConfig struct {
	MaxRunsVisible int `yaml:"max_runs_visible" json:"max_runs_visible"`
	RefreshMS      int `yaml:"refresh_ms" json:"refresh_ms"`
}

type NotificationsConfig struct {
	Enabled    bool `yaml:"enabled" json:"enabled"`
	OnWarn     bool `yaml:"on_warn" json:"on_warn"`
	RateLimitS int  `yaml:"rate_limit_s" json:"rate_limit_s"`
}

// Load reads configuration from path. Missing file returns defaults.
func Load(path string) (*Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config parse error: %w", err)
	}
	return cfg, nil
}

// LoadFromPlatform loads config from the default platform path.
func LoadFromPlatform() (*Config, *platform.Paths, error) {
	paths, err := platform.Resolve()
	if err != nil {
		return nil, nil, err
	}
	cfg, err := Load(paths.ConfigPath)
	if err != nil {
		return nil, nil, err
	}
	if cfg.Daemon.DataDir != "" {
		cfg.Daemon.DataDir = platform.ExpandHome(cfg.Daemon.DataDir)
	} else {
		cfg.Daemon.DataDir = paths.DataDir
	}
	if cfg.Daemon.SocketPath == "" {
		cfg.Daemon.SocketPath = paths.SocketPath
	} else {
		cfg.Daemon.SocketPath = platform.ExpandHome(cfg.Daemon.SocketPath)
	}

	// Expand watch paths for every platform.
	for _, pc := range cfg.Platforms.byHarness() {
		pc.TranscriptWatchPath = platform.ExpandHome(pc.TranscriptWatchPath)
	}
	return cfg, paths, nil
}

// Save writes configuration to disk.
func (c *Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Set updates a dotted config key.
func (c *Config) Set(key, value string) error {
	switch key {
	case "analytics.enabled":
		c.Analytics.Enabled = value == "true"
	case "telemetry.enabled":
		c.Telemetry.Enabled = value == "true"
	case "telemetry.share_by_default":
		c.Telemetry.ShareByDefault = value == "true"
	case "telemetry.consent_shown":
		c.Telemetry.ConsentShown = value == "true"
	case "retention.days", "retention.max_days":
		v, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		c.Retention.MaxDays = v
	case "notifications.enabled":
		c.Notifications.Enabled = value == "true"
	case "notifications.on_warn":
		c.Notifications.OnWarn = value == "true"
	case "notifications.rate_limit_s":
		v, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		c.Notifications.RateLimitS = v
	default:
		if c.Verification.ShellVerifier.AllowRerun == nil {
			c.Verification.ShellVerifier.AllowRerun = make(map[string]bool)
		}
		prefix := "shell_verifier.allow_rerun."
		if strings.HasPrefix(key, prefix) {
			path := key[len(prefix):]
			c.Verification.ShellVerifier.AllowRerun[path] = value == "true"
			return nil
		}
		// platforms.<name>.enabled
		if strings.HasPrefix(key, "platforms.") && strings.HasSuffix(key, ".enabled") {
			name := strings.TrimSuffix(strings.TrimPrefix(key, "platforms."), ".enabled")
			pc, ok := c.Platforms.byHarness()[name]
			if !ok {
				return fmt.Errorf("unknown platform: %s", name)
			}
			pc.Enabled = value == "true"
			return nil
		}
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

// Get returns a config value by dotted key.
func (c *Config) Get(key string) (string, error) {
	switch key {
	case "analytics.enabled":
		return fmt.Sprintf("%v", c.Analytics.Enabled), nil
	case "telemetry.enabled":
		return fmt.Sprintf("%v", c.Telemetry.Enabled), nil
	case "telemetry.share_by_default":
		return fmt.Sprintf("%v", c.Telemetry.ShareByDefault), nil
	case "telemetry.consent_shown":
		return fmt.Sprintf("%v", c.Telemetry.ConsentShown), nil
	case "retention.max_days":
		return fmt.Sprintf("%d", c.Retention.MaxDays), nil
	case "notifications.enabled":
		return fmt.Sprintf("%v", c.Notifications.Enabled), nil
	case "notifications.on_warn":
		return fmt.Sprintf("%v", c.Notifications.OnWarn), nil
	case "notifications.rate_limit_s":
		return fmt.Sprintf("%d", c.Notifications.RateLimitS), nil
	default:
		prefix := "shell_verifier.allow_rerun."
		if strings.HasPrefix(key, prefix) {
			path := key[len(prefix):]
			if c.Verification.ShellVerifier.AllowRerun != nil {
				return fmt.Sprintf("%v", c.Verification.ShellVerifier.AllowRerun[path]), nil
			}
			return "false", nil
		}
		if strings.HasPrefix(key, "platforms.") && strings.HasSuffix(key, ".enabled") {
			name := strings.TrimSuffix(strings.TrimPrefix(key, "platforms."), ".enabled")
			pc, ok := c.Platforms.byHarness()[name]
			if !ok {
				return "", fmt.Errorf("unknown platform: %s", name)
			}
			return fmt.Sprintf("%v", pc.Enabled), nil
		}
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}
