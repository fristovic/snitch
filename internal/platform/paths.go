package platform

import (
	"os"
	"path/filepath"
)

// Paths holds platform-resolved paths for Snitch data.
type Paths struct {
	DataDir    string
	SocketPath string
	LogPath    string
	ConfigPath string
	DBPath     string
}

// Resolve returns platform-specific paths, creating the data directory if needed.
func Resolve() (*Paths, error) {
	p := defaultPaths()
	if err := os.MkdirAll(p.DataDir, 0o700); err != nil {
		return nil, err
	}
	return p, nil
}

// ExpandHome replaces leading ~ with the user's home directory.
func ExpandHome(path string) string {
	if len(path) == 0 {
		return path
	}
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if len(path) == 1 {
			return home
		}
		if path[1] == '/' || path[1] == '\\' {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
