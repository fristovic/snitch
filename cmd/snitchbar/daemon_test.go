package main

import (
	"testing"
	"time"
)

func TestWaitForSocketTimeout(t *testing.T) {
	if waitForSocket("/tmp/snitch-nonexistent.sock", 50*time.Millisecond) {
		t.Fatal("expected timeout")
	}
}
