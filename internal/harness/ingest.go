package harness

import (
	"log/slog"
	"sort"

	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/event"
)

// StartIngestion starts every enabled harness's transcript source via its
// descriptor. Returns all stoppers for graceful shutdown.
func StartIngestion(bus *event.Bus, cfg *config.Config, reg *Registry) ([]Stopper, error) {
	descriptors := reg.All()
	sort.Slice(descriptors, func(i, j int) bool {
		return descriptors[i].Name < descriptors[j].Name
	})

	var stoppers []Stopper
	for _, d := range descriptors {
		pc, ok := cfg.Platforms.ForHarness(d.Name)
		if !ok || !pc.Enabled {
			continue
		}
		if pc.TranscriptWatchPath == "" {
			slog.Warn("platform enabled but watch path empty", "harness", d.Name)
			continue
		}
		s, err := d.Ingest(bus, pc.TranscriptWatchPath)
		if err != nil {
			slog.Warn("harness ingestion start failed", "harness", d.Name, "err", err)
			continue
		}
		stoppers = append(stoppers, s)
	}
	return stoppers, nil
}
