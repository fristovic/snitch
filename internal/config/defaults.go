package config

// Default returns a Config with all defaults.
func Default() *Config {
	return &Config{
		Daemon: DaemonConfig{
			DataDir:  "~/.snitch",
			LogLevel: "info",
		},
		Cursor: CursorConfig{
			Enabled:             true,
			TranscriptWatchPath: "~/.cursor/projects",
		},
		Verification: VerificationConfig{
			MaxConcurrentVerifications: 3,
		},
		Analytics: AnalyticsConfig{
			Enabled:   false,
			Endpoint:  "https://analytics.snitch.dev/v1/report",
			IntervalH: 24,
		},
		Retention: RetentionConfig{
			MaxDays:      30,
			KeepFailures: true,
		},
		Display: DisplayConfig{
			TUI: TUIConfig{MaxRunsVisible: 100, RefreshMS: 500},
		},
	}
}
