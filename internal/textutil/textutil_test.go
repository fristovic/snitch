package textutil

import "testing"

func TestTruncateRunes(t *testing.T) {
	if got := TruncateRunes("hello", 10); got != "hello" {
		t.Fatalf("got %q", got)
	}
	if got := TruncateRunes("hello world", 8); got != "hello w…" {
		t.Fatalf("got %q", got)
	}
}

func TestOneLine(t *testing.T) {
	in := "line one\n\tline two"
	if got := OneLine(in, 20); got != "line one line two" {
		t.Fatalf("got %q", got)
	}
}
