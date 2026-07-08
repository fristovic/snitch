package config

// Default returns a Config with all defaults.
func Default() *Config {
	return &Config{
		Daemon: DaemonConfig{
			DataDir:  "~/.snitch",
			LogLevel: "info",
		},
		Platforms: PlatformsConfig{
			// Cursor is on by default; all other harnesses are opt-in so existing
			// users see no behavior change on upgrade.
			Cursor: PlatformConfig{
				Enabled:             true,
				TranscriptWatchPath: "~/.cursor/projects",
			},
			Claude:   PlatformConfig{Enabled: false, TranscriptWatchPath: "~/.claude/projects"},
			Codex:    PlatformConfig{Enabled: false, TranscriptWatchPath: "~/.codex/sessions"},
			Pi:       PlatformConfig{Enabled: false, TranscriptWatchPath: "~/.pi/agent/sessions"},
			OpenCode: PlatformConfig{Enabled: false, TranscriptWatchPath: "~/.local/share/opencode/opencode.db"},
		},
		Verification: VerificationConfig{
			MaxConcurrentVerifications: 3,
		},
		Analytics: AnalyticsConfig{
			Enabled:   false,
			Endpoint:  "https://telemetry.snitchworks.com/api/v1/analytics/report",
			IntervalH: 24,
		},
		Retention: RetentionConfig{
			MaxDays:      30,
			KeepFailures: true,
		},
		Display: DisplayConfig{
			TUI: TUIConfig{MaxRunsVisible: 100, RefreshMS: 500},
		},
		Notifications: NotificationsConfig{
			Enabled:    true,
			OnWarn:     false,
			RateLimitS: 5,
		},
		Telemetry: TelemetryConfig{
			// Telemetry is off by default — sharing labeled verdicts is opt-in.
			Enabled:   false,
			IntervalM: 60,
		},
	}
}
