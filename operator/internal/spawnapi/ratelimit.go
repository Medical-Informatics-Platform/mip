package spawnapi

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// namespaceLimiter is a token bucket per federation namespace. It is consulted
// only after authorization, so its keys are bounded by the namespaces whose hub
// holds a valid token.
type namespaceLimiter struct {
	limit rate.Limit
	burst int

	mu       sync.Mutex
	limiters map[string]*rate.Limiter
}

func newNamespaceLimiter(limit rate.Limit, burst int) *namespaceLimiter {
	return &namespaceLimiter{limit: limit, burst: burst, limiters: map[string]*rate.Limiter{}}
}

// allow reports whether a request may go ahead, and if not, when to retry.
func (l *namespaceLimiter) allow(ns string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	lim, ok := l.limiters[ns]
	if !ok {
		lim = rate.NewLimiter(l.limit, l.burst)
		l.limiters[ns] = lim
	}
	l.mu.Unlock()

	r := lim.ReserveN(now, 1)
	if !r.OK() {
		return false, time.Second
	}
	if d := r.DelayFrom(now); d > 0 {
		r.CancelAt(now)
		return false, d
	}
	return true, 0
}
