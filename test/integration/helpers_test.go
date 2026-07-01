package integration_test

import (
	"os"
	"path/filepath"
	"testing"
)

func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	_ = os.Mkdir(filepath.Join(dir, ".git"), 0o700)
	return dir
}
