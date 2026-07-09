package main

import (
	"strings"
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

func TestLatestLiePreview(t *testing.T) {
	if got := LatestLiePreview(nil); got != "No lies yet" {
		t.Fatalf("got %q", got)
	}
	got := LatestLiePreview(&record.LieClaim{
		Claim: record.Claim{ClaimType: "test_pass", Claimed: "All tests pass."},
	})
	if got != `Latest: test_pass — "All tests pass."` {
		t.Fatalf("got %q", got)
	}
	long := LatestLiePreview(&record.LieClaim{
		Claim: record.Claim{
			ClaimType: "committed",
			Claimed:   "I committed the changes to the repository after fixing everything carefully",
		},
	})
	if !strings.HasPrefix(long, "Latest: committed — \"") {
		t.Fatalf("got %q", long)
	}
	if !strings.Contains(long, "…") {
		t.Fatalf("expected truncation: %q", long)
	}
}
