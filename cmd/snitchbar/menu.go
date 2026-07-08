package main

import (
	"github.com/fristovic/snitch/internal/record"
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
