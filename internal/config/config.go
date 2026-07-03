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
	Daemon       DaemonConfig       `yaml:"daemon"`
	Cursor       CursorConfig       `yaml:"cursor"`
	Verification VerificationConfig `yaml:"verification"`
	Analytics    AnalyticsConfig    `yaml:"analytics"`
	Retention    RetentionConfig    `yaml:"retention"`
	Display      DisplayConfig      `yaml:"display"`
	Notifications NotificationsConfig `yaml:"notifications"`
}

type DaemonConfig struct {
	DataDir    string `yaml:"data_dir"`
	SocketPath string `yaml:"socket_path"`
	LogLevel   string `yaml:"log_level"`
}

type CursorConfig struct {
	Enabled             bool   `yaml:"enabled"`
	TranscriptWatchPath string `yaml:"transcript_watch_path"`
}

type VerificationConfig struct {
	MaxConcurrentVerifications int                `yaml:"max_concurrent_verifications"`
	ShellVerifier              ShellVerifierConfig `yaml:"shell_verifier"`
}

type ShellVerifierConfig struct {
	AllowRerun map[string]bool `yaml:"allow_rerun"`
}

type AnalyticsConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Endpoint       string `yaml:"endpoint"`
	IntervalH      int    `yaml:"interval_h"`
	SigningKeyPath string `yaml:"signing_key_path"`
}

type RetentionConfig struct {
	MaxDays      int  `yaml:"max_days"`
	KeepFailures bool `yaml:"keep_failures"`
}

type DisplayConfig struct {
	TUI TUIConfig `yaml:"tui"`
}

type TUIConfig struct {
	MaxRunsVisible int `yaml:"max_runs_visible"`
	RefreshMS      int `yaml:"refresh_ms"`
}

type NotificationsConfig struct {
	Enabled    bool `yaml:"enabled"`
	OnWarn     bool `yaml:"on_warn"`
	RateLimitS int  `yaml:"rate_limit_s"`
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
	cfg.Cursor.TranscriptWatchPath = platform.ExpandHome(cfg.Cursor.TranscriptWatchPath)
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
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

// Get returns a config value by dotted key.
func (c *Config) Get(key string) (string, error) {
	switch key {
	case "analytics.enabled":
		return fmt.Sprintf("%v", c.Analytics.Enabled), nil
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
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}
