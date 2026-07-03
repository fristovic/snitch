package notify

import (
	"testing"
	"time"

	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/record"
)

func TestFormatNotification(t *testing.T) {
	title, body := FormatNotification("test_pass", "all tests pass", "no test command ran", "/Users/me/proj")
	if title != "Snitch — test_pass (proj)" {
		t.Fatalf("title: %q", title)
	}
	if body != `"all tests pass" → no test command ran` {
		t.Fatalf("body: %q", body)
	}
}

func TestTopLieClaim(t *testing.T) {
	claims := []record.Claim{
		{ClaimType: "stub", Verified: 1, Severity: 1},
		{ClaimType: "test_pass", Verified: -1, Severity: 3},
		{ClaimType: "committed", Verified: -1, Severity: 2},
	}
	top := TopLieClaim(claims)
	if top == nil || top.ClaimType != "test_pass" {
		t.Fatalf("got %+v", top)
	}
}

func TestRateLimiter(t *testing.T) {
	resetLimiter()
	if !limiter.allow("run-1", time.Second) {
		t.Fatal("first should pass")
	}
	if limiter.allow("run-1", time.Second) {
		t.Fatal("duplicate run should be blocked")
	}
	if limiter.allow("run-2", time.Second) {
		t.Fatal("global gap should block different run")
	}
}

func TestMaybeNotifyDisabled(t *testing.T) {
	resetLimiter()
	MaybeNotify(nil, event.RunVerifiedPayload{Verdict: record.VerdictFail}, config.NotificationsConfig{Enabled: false})
}
