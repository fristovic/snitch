package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fristovic/snitch/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.Default()
	if !cfg.Cursor.Enabled {
		t.Fatal("cursor watcher should be enabled by default")
	}
	if cfg.Display.TUI.MaxRunsVisible != 100 {
		t.Fatalf("expected max runs 100, got %d", cfg.Display.TUI.MaxRunsVisible)
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, err := config.Load("/nonexistent/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Retention.MaxDays != 30 {
		t.Fatalf("expected 30 day retention, got %d", cfg.Retention.MaxDays)
	}
}

func TestInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	_ = os.WriteFile(path, []byte(":\n  bad: ["), 0o644)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestSetGet(t *testing.T) {
	cfg := config.Default()
	if err := cfg.Set("analytics.enabled", "true"); err != nil {
		t.Fatal(err)
	}
	val, err := cfg.Get("analytics.enabled")
	if err != nil || val != "true" {
		t.Fatalf("got %s err %v", val, err)
	}
}
