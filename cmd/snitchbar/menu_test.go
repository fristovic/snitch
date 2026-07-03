package main

import (
	"testing"

	"github.com/fristovic/snitch/internal/record"
)

func TestStatusLabel(t *testing.T) {
	if got := StatusLabel(MenuState{Connected: true}); got != "Snitching..." {
		t.Fatalf("got %q", got)
	}
	if got := StatusLabel(MenuState{Connected: true, Alert: true}); got != "Snitching..." {
		t.Fatalf("got %q", got)
	}
	if got := StatusLabel(MenuState{Paused: true}); got != "Paused" {
		t.Fatalf("got %q", got)
	}
	if got := StatusLabel(MenuState{Starting: true}); got != "Starting…" {
		t.Fatalf("got %q", got)
	}
	if got := StatusLabel(MenuState{}); got != "Offline" {
		t.Fatalf("got %q", got)
	}
}

func TestToggleLabel(t *testing.T) {
	if got := ToggleLabel(MenuState{Connected: true}); got != "Stop Snitching" {
		t.Fatalf("got %q", got)
	}
	if got := ToggleLabel(MenuState{Paused: true}); got != "Start Snitching" {
		t.Fatalf("got %q", got)
	}
	if got := ToggleLabel(MenuState{Starting: true}); got != "Starting…" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatLieCopy(t *testing.T) {
	got := FormatLieCopy(record.LieClaim{Claim: record.Claim{
		ClaimType: "stub",
		Claimed:   "fully implemented",
		Actual:    "placeholder",
	}})
	want := "stub\n\"fully implemented\"\n→ placeholder"
	if got != want {
		t.Fatalf("got %q", got)
	}
}
