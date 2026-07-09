package notify

import (
	"testing"
	"time"

	"github.com/fristovic/snitch/internal/claims"
	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/record"
)

func TestFormatNotificationFields(t *testing.T) {
	title, body := FormatNotificationFields(claims.DisplayFields{
		ClaimType: "test_pass",
		Claimed:   "all tests pass",
		Actual:    "no test command ran",
	}, "/Users/me/proj")
	if title != "Snitch — Tests passed (proj)" {
		t.Fatalf("title: %q", title)
	}
	if body != `"all tests pass" → no test command ran` {
		t.Fatalf("body: %q", body)
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

func TestDeliverDisabled(t *testing.T) {
	resetLimiter()
	Deliver(event.RunVerifiedPayload{
		Verdict:  record.VerdictFail,
		TopClaim: &event.TopFalseClaim{ClaimType: "test_pass"},
	}, config.NotificationsConfig{Enabled: false})
}

func TestDeliverNoTopClaim(t *testing.T) {
	resetLimiter()
	Deliver(event.RunVerifiedPayload{
		Verdict: record.VerdictFail,
	}, config.NotificationsConfig{Enabled: true})
}

func TestDeliverWarnGated(t *testing.T) {
	resetLimiter()
	Deliver(event.RunVerifiedPayload{
		Verdict:  record.VerdictWarn,
		TopClaim: &event.TopFalseClaim{ClaimType: "test_pass"},
	}, config.NotificationsConfig{Enabled: true, OnWarn: false})
}
