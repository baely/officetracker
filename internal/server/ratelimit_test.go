package server

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	context2 "github.com/baely/officetracker/internal/context"
)

func TestClientIP(t *testing.T) {
	cases := []struct {
		name       string
		xff        string
		remoteAddr string
		want       string
	}{
		{"xff single", "203.0.113.7", "10.0.0.1:5000", "203.0.113.7"},
		{"xff rightmost trusted", "1.1.1.1, 2.2.2.2, 3.3.3.3", "10.0.0.1:5000", "3.3.3.3"},
		{"xff trims spaces", "1.1.1.1,  2.2.2.2 ", "10.0.0.1:5000", "2.2.2.2"},
		{"no xff, host:port", "", "192.168.1.5:41234", "192.168.1.5"},
		{"no xff, malformed addr", "", "not-an-addr", "not-an-addr"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = c.remoteAddr
			if c.xff != "" {
				r.Header.Set("X-Forwarded-For", c.xff)
			}
			if got := clientIP(r); got != c.want {
				t.Errorf("clientIP = %q, want %q", got, c.want)
			}
		})
	}
}

// A request with an authenticated user is keyed and limited per user; anything
// else is keyed and limited per client IP.
func TestRateLimiterClient(t *testing.T) {
	authed := rateLimits{perMinute: 120, perHour: 1200}
	unauthed := rateLimits{perMinute: 10, perHour: 10}
	rl := newRateLimiter(nil, authed, unauthed)

	t.Run("authenticated -> per user", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		val := context2.CtxValue{}
		val.Set(context2.CtxUserIDKey, 5)
		r = r.WithContext(context.WithValue(r.Context(), context2.CtxKey, val))

		key, limits := rl.client(r)
		if key != "user:5" {
			t.Errorf("key = %q, want user:5", key)
		}
		if limits != authed {
			t.Errorf("limits = %+v, want authed", limits)
		}
	})

	t.Run("anonymous -> per ip", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = "192.168.0.9:1234"
		key, limits := rl.client(r)
		if key != "ip:192.168.0.9" {
			t.Errorf("key = %q, want ip:192.168.0.9", key)
		}
		if limits != unauthed {
			t.Errorf("limits = %+v, want unauthed", limits)
		}
	})
}

// The in-memory limiter allows a burst up to the minute allowance, then denies
// with a positive retry-after. Time is injected, so the test is deterministic.
func TestAllowInMemoryMinuteBurst(t *testing.T) {
	rl := newRateLimiter(nil, rateLimits{}, rateLimits{})
	limits := rateLimits{perMinute: 5, perHour: 1000}
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 5; i++ {
		ok, _ := rl.allowInMemory("k", limits, now)
		if !ok {
			t.Fatalf("request %d should be allowed within the burst", i+1)
		}
	}

	ok, retryAfter := rl.allowInMemory("k", limits, now)
	if ok {
		t.Fatal("6th request should be denied once the minute burst is spent")
	}
	if retryAfter <= 0 {
		t.Errorf("denied request retry-after = %v, want > 0", retryAfter)
	}

	// After enough time for one token to refill (rate = 5/60 per sec => 12s), a
	// single request is allowed again.
	later := now.Add(12 * time.Second)
	if ok, _ := rl.allowInMemory("k", limits, later); !ok {
		t.Error("request should be allowed after the bucket refills")
	}
}

// The hour bucket caps sustained usage independently of the minute bucket.
func TestAllowInMemoryHourCap(t *testing.T) {
	rl := newRateLimiter(nil, rateLimits{}, rateLimits{})
	limits := rateLimits{perMinute: 1000, perHour: 3}
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 3; i++ {
		if ok, _ := rl.allowInMemory("h", limits, now); !ok {
			t.Fatalf("request %d should be within the hourly cap", i+1)
		}
	}
	if ok, retryAfter := rl.allowInMemory("h", limits, now); ok || retryAfter <= 0 {
		t.Errorf("4th request should be denied by the hour cap (ok=%v, retry=%v)", ok, retryAfter)
	}
}

// A denied request returns its tokens, so it does not deplete the buckets
// further: exactly one refilled token yields exactly one more allowed request.
func TestDeniedRequestDoesNotConsume(t *testing.T) {
	rl := newRateLimiter(nil, rateLimits{}, rateLimits{})
	limits := rateLimits{perMinute: 2, perHour: 1000}
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)

	// Spend the burst.
	rl.allowInMemory("k", limits, now)
	rl.allowInMemory("k", limits, now)
	// Several denials while empty.
	for i := 0; i < 3; i++ {
		if ok, _ := rl.allowInMemory("k", limits, now); ok {
			t.Fatal("expected denial while bucket empty")
		}
	}
	// Refill exactly one token (rate 2/60 per sec => 30s per token).
	if ok, _ := rl.allowInMemory("k", limits, now.Add(30*time.Second)); !ok {
		t.Error("one token should have refilled despite the intervening denials")
	}
}

// Idle clients are evicted during the periodic sweep.
func TestLimiterEvictsIdleClients(t *testing.T) {
	rl := newRateLimiter(nil, rateLimits{}, rateLimits{})
	limits := rateLimits{perMinute: 10, perHour: 100}
	t0 := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)

	rl.allowInMemory("old", limits, t0)
	if _, ok := rl.clients["old"]; !ok {
		t.Fatal("client 'old' should be tracked")
	}

	// Two hours later a new client triggers a sweep; 'old' is idle > 1h and evicted.
	rl.allowInMemory("new", limits, t0.Add(2*time.Hour))
	if _, ok := rl.clients["old"]; ok {
		t.Error("idle client 'old' should have been evicted")
	}
	if _, ok := rl.clients["new"]; !ok {
		t.Error("active client 'new' should be tracked")
	}
}
