package ipc

import (
	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/platform"
)

// ResolveSocket returns the IPC socket path from an explicit override or platform defaults.
func ResolveSocket(override string) string {
	if override != "" {
		return override
	}
	paths, err := platform.Resolve()
	if err != nil {
		return ""
	}
	cfg, err := config.Load(paths.ConfigPath)
	if err != nil {
		return paths.SocketPath
	}
	if cfg.Daemon.SocketPath != "" {
		return cfg.Daemon.SocketPath
	}
	return paths.SocketPath
}
