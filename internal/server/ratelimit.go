package server

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/baely/officetracker/internal/database"
)

type rateLimits struct {
	perMinute int // short-window burst allowance
	perHour   int // sustained hourly cap
}

// Limits applied per client. Authenticated requests are limited per user;
// everything else per client IP, on a much tighter budget.
var (
	authedRateLimits = rateLimits{perMinute: 120, perHour: 1200}
	// A flat hourly cap for unauthenticated clients: the minute bucket
	// matches the hour bucket, so the whole allowance may be spent in a
	// single burst.
	unauthedRateLimits = rateLimits{perMinute: 120, perHour: 120}
)

const (
	// After an hour idle both buckets are full again, so an evicted entry is
	// indistinguishable from a fresh one.
	limiterIdleEviction = time.Hour
	limiterSweepEvery   = 5 * time.Minute
)

// clientLimiter holds the two in-memory token buckets for a single client. A
// request must claim a token from both: the minute bucket permits short
// bursts while the hour bucket caps sustained usage.
type clientLimiter struct {
	minute   *rate.Limiter
	hour     *rate.Limiter
	lastSeen time.Time
}

// rateLimiter enforces per-client limits. With Redis available the buckets
// live there, shared across all instances; without it (standalone mode) each
// instance keeps its own in-memory buckets.
type rateLimiter struct {
	authed   rateLimits
	unauthed rateLimits

	redis *database.Redis // nil in standalone mode

	// In-memory fallback state, used only when redis is nil.
	mu        sync.Mutex
	clients   map[string]*clientLimiter
	lastSweep time.Time
}

func newRateLimiter(redis *database.Redis, authed, unauthed rateLimits) *rateLimiter {
	return &rateLimiter{
		authed:   authed,
		unauthed: unauthed,
		redis:    redis,
		clients:  make(map[string]*clientLimiter),
	}
}

func (rl *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, limits := rl.client(r)
		ok, retryAfter := rl.allow(r.Context(), key, limits, time.Now())
		if !ok {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
			writeError(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// client identifies the requester and the limits that apply to it:
// authenticated requests are limited per user, everything else per client IP.
func (rl *rateLimiter) client(r *http.Request) (string, rateLimits) {
	if userID, err := getUserID(r); err == nil && userID != 0 {
		return "user:" + strconv.Itoa(userID), rl.authed
	}
	return "ip:" + clientIP(r), rl.unauthed
}

// allow reports whether the client identified by key may proceed at time now.
// When denied it also returns how long the client should wait before retrying.
func (rl *rateLimiter) allow(ctx context.Context, key string, limits rateLimits, now time.Time) (bool, time.Duration) {
	if rl.redis == nil {
		return rl.allowInMemory(key, limits, now)
	}
	ok, retryAfter, err := rl.redis.RateLimitAllow(ctx, "ratelimit:"+key, limits.perMinute, limits.perHour, now)
	if err != nil {
		// Fail open: a Redis outage shouldn't take the site down.
		slog.Error(fmt.Sprintf("rate limit check failed, allowing request: %v", err))
		return true, 0
	}
	return ok, retryAfter
}

func (rl *rateLimiter) allowInMemory(key string, limits rateLimits, now time.Time) (bool, time.Duration) {
	c := rl.limiterFor(key, limits, now)

	minuteRes := c.minute.ReserveN(now, 1)
	hourRes := c.hour.ReserveN(now, 1)
	minuteDelay := minuteRes.DelayFrom(now)
	hourDelay := hourRes.DelayFrom(now)
	if minuteDelay == 0 && hourDelay == 0 {
		return true, 0
	}

	// Denied: return both claimed tokens so the rejected attempt doesn't
	// count against either limit.
	minuteRes.CancelAt(now)
	hourRes.CancelAt(now)
	return false, max(minuteDelay, hourDelay)
}

func (rl *rateLimiter) limiterFor(key string, limits rateLimits, now time.Time) *clientLimiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if now.Sub(rl.lastSweep) >= limiterSweepEvery {
		rl.lastSweep = now
		for k, c := range rl.clients {
			if now.Sub(c.lastSeen) >= limiterIdleEviction {
				delete(rl.clients, k)
			}
		}
	}

	c, ok := rl.clients[key]
	if !ok {
		c = &clientLimiter{
			minute: rate.NewLimiter(rate.Limit(float64(limits.perMinute)/60), limits.perMinute),
			hour:   rate.NewLimiter(rate.Limit(float64(limits.perHour)/3600), limits.perHour),
		}
		rl.clients[key] = c
	}
	c.lastSeen = now
	return c
}

// clientIP returns the requesting client's IP. On Cloud Run the rightmost
// X-Forwarded-For entry is appended by Google's front end and is trustworthy;
// with no proxy in front the header is absent and RemoteAddr is used.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
