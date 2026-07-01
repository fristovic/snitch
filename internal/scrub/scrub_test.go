package scrub

import "testing"

func TestScrubAPIKeyFlag(t *testing.T) {
	in := "cursor agent --api-key sk-abcdefghijklmnopqrstuvwxyz123456"
	out := Scrub(in)
	if out == in {
		t.Fatal("expected redaction")
	}
	if contains(out, "sk-abcdefghijklmnopqrstuvwxyz123456") {
		t.Fatalf("secret leaked: %s", out)
	}
}

func TestScrubBearer(t *testing.T) {
	in := "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	out := Scrub(in)
	if contains(out, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9") {
		t.Fatalf("bearer leaked: %s", out)
	}
}

func TestScrubEnvAssignment(t *testing.T) {
	in := "OPENAI_API_KEY=sk-abcdefghijklmnopqrstuvwxyz1234567890"
	out := Scrub(in)
	if contains(out, "sk-abcdefghijklmnopqrstuvwxyz1234567890") {
		t.Fatalf("env secret leaked: %s", out)
	}
}

func TestScrubPreservesSafe(t *testing.T) {
	in := "created file hello.py successfully"
	if Scrub(in) != in {
		t.Fatalf("safe text modified: %s", Scrub(in))
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
