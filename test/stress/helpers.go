package stress

import (
	"os"
	"path/filepath"

	"github.com/fristovic/snitch/internal/event"
)

func writeFile(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

func newTestBus() *event.Bus {
	return event.NewBus()
}
