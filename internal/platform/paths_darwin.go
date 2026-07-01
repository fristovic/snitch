//go:build darwin

package platform

import (
	"os"
	"path/filepath"
)

func defaultPaths() *Paths {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".snitch")
	return &Paths{
		DataDir:    dataDir,
		SocketPath: filepath.Join(dataDir, "snitch.sock"),
		LogPath:    filepath.Join(dataDir, "snitchd.log"),
		ConfigPath: filepath.Join(dataDir, "config.yaml"),
		DBPath:     filepath.Join(dataDir, "snitch.db"),
	}
}
