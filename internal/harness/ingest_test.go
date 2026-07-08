package harness

import (
	"testing"

	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/event"
)

func TestStartIngestionStartsOnlyEnabledHarnesses(t *testing.T) {
	bus := event.NewBus()
	defer bus.Close()

	cfg := config.Default()
	cfg.Platforms.Cursor = config.PlatformConfig{Enabled: true, TranscriptWatchPath: t.TempDir()}
	cfg.Platforms.Claude = config.PlatformConfig{Enabled: true, TranscriptWatchPath: t.TempDir()}
	cfg.Platforms.Codex = config.PlatformConfig{Enabled: false, TranscriptWatchPath: t.TempDir()}
	cfg.Platforms.Pi = config.PlatformConfig{Enabled: false}
	// Enabled but the DB does not exist: reader start fails and is skipped
	// with a warning rather than aborting the daemon.
	cfg.Platforms.OpenCode = config.PlatformConfig{Enabled: false, TranscriptWatchPath: "/nonexistent/opencode.db"}

	stoppers, err := StartIngestion(bus, cfg, NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, s := range stoppers {
			_ = s.Stop()
		}
	}()
	if len(stoppers) != 2 {
		t.Fatalf("expected 2 stoppers (cursor+claude), got %d", len(stoppers))
	}
}

func TestStartIngestionSkipsEmptyWatchPath(t *testing.T) {
	bus := event.NewBus()
	defer bus.Close()

	cfg := config.Default()
	cfg.Platforms.Cursor = config.PlatformConfig{Enabled: true, TranscriptWatchPath: ""}
	cfg.Platforms.Claude = config.PlatformConfig{Enabled: false}
	cfg.Platforms.Codex = config.PlatformConfig{Enabled: false}
	cfg.Platforms.Pi = config.PlatformConfig{Enabled: false}
	cfg.Platforms.OpenCode = config.PlatformConfig{Enabled: false}

	stoppers, err := StartIngestion(bus, cfg, NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if len(stoppers) != 0 {
		t.Fatalf("expected no stoppers, got %d", len(stoppers))
	}
}

func TestRegistryShellResolver(t *testing.T) {
	reg := NewRegistry()
	if reg.ShellResolver("cursor") == nil {
		t.Fatal("cursor should have a terminal-file shell resolver")
	}
	if reg.ShellResolver("claude") == nil {
		t.Fatal("claude should have the noop resolver, not nil")
	}
	if reg.ShellResolver("nope") != nil {
		t.Fatal("unknown harness should resolve to nil")
	}
	if reg.ShellResolver("") != nil {
		t.Fatal("empty harness should resolve to nil")
	}
}
