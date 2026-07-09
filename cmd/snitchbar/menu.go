package main

import (
	"fmt"
	"strings"

	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/textutil"
)

// MenuState drives the systray dropdown labels.
type MenuState struct {
	Connected bool
	Paused    bool
	Starting  bool
	Alert     bool
	Lie       *record.LieClaim
}

// StatusLabel is the non-clickable status row at the top of the menu.
func StatusLabel(s MenuState) string {
	if s.Paused {
		return "Paused"
	}
	if s.Starting {
		return "Starting…"
	}
	if !s.Connected {
		return "Offline"
	}
	return "Snitching..."
}

// ToggleLabel is the start/stop action based on daemon state.
func ToggleLabel(s MenuState) string {
	if s.Starting {
		return "Starting…"
	}
	if s.Paused || !s.Connected {
		return "Start Snitching"
	}
	return "Stop Snitching"
}

// LatestLiePreview is a disabled context row for the latest caught lie.
// Empty string means "No lies yet".
func LatestLiePreview(lie *record.LieClaim) string {
	if lie == nil {
		return "No lies yet"
	}
	claimed := strings.Join(strings.Fields(lie.Claimed), " ")
	claimed = textutil.TruncateRunes(claimed, 42)
	if claimed == "" {
		claimed = lie.ClaimType
	}
	return fmt.Sprintf("Latest: %s — %q", lie.ClaimType, claimed)
}
