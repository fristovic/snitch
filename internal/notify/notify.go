package notify

import (
	"sync"
	"time"

	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/event"
	"github.com/fristovic/snitch/internal/record"
)

var limiter = &rateLimiter{}

// MaybeNotify sends a macOS notification for a verified run when configured.
func MaybeNotify(store *record.Store, p event.RunVerifiedPayload, cfg config.NotificationsConfig) {
	if !cfg.Enabled {
		return
	}
	switch p.Verdict {
	case record.VerdictFail:
	case record.VerdictWarn:
		if !cfg.OnWarn {
			return
		}
	default:
		return
	}

	claims, err := store.GetClaimsByRun(p.RunID)
	if err != nil {
		return
	}
	top := TopLieClaim(claims)
	if top == nil {
		return
	}

	rate := time.Duration(cfg.RateLimitS) * time.Second
	if rate <= 0 {
		rate = 5 * time.Second
	}
	if !limiter.allow(p.RunID, rate) {
		return
	}

	title, body := FormatNotification(top.ClaimType, top.Claimed, top.Actual, p.ProjectPath)
	_ = Notify(title, body)
}

type rateLimiter struct {
	mu      sync.Mutex
	lastAny time.Time
	byRun   map[string]time.Time
}

func (r *rateLimiter) allow(runID string, minGap time.Duration) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	if r.byRun == nil {
		r.byRun = make(map[string]time.Time)
	}
	if last, ok := r.byRun[runID]; ok && now.Sub(last) < minGap {
		return false
	}
	if !r.lastAny.IsZero() && now.Sub(r.lastAny) < minGap {
		return false
	}
	r.byRun[runID] = now
	r.lastAny = now
	return true
}

// resetLimiter clears rate-limit state (tests only).
func resetLimiter() {
	limiter = &rateLimiter{}
}
