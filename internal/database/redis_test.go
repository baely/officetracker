package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/baely/officetracker/internal/config"
)

// Redis integration tests run against a real Redis. They are skipped unless
// REDIS_TEST_ADDR is set (host:port), so the default `go test ./...` stays green
// without a Redis. CI provides a redis service container. Locally:
//
//	docker run -d -p 6379:6379 redis:7-alpine
//	REDIS_TEST_ADDR=localhost:6379 go test ./internal/database/...
func redisTestClient(t *testing.T) *Redis {
	t.Helper()
	addr := os.Getenv("REDIS_TEST_ADDR")
	if addr == "" {
		t.Skip("REDIS_TEST_ADDR not set; skipping Redis integration tests")
	}
	r, err := NewRedis(config.Redis{Host: addr})
	if err != nil {
		t.Fatalf("NewRedis: %v", err)
	}
	return r
}

func TestRedisStateRoundTrip(t *testing.T) {
	r := redisTestClient(t)
	ctx := context.Background()
	key := "test:state:roundtrip"
	_ = r.DeleteState(ctx, key)

	if err := r.SetState(ctx, key, 4321, time.Minute); err != nil {
		t.Fatalf("SetState: %v", err)
	}
	got, err := r.GetStateInt(ctx, key)
	if err != nil {
		t.Fatalf("GetStateInt: %v", err)
	}
	if got != 4321 {
		t.Errorf("GetStateInt = %d, want 4321", got)
	}

	if err := r.DeleteState(ctx, key); err != nil {
		t.Fatalf("DeleteState: %v", err)
	}
	if _, err := r.GetStateInt(ctx, key); err == nil {
		t.Error("expected an error reading a deleted key")
	}
}

// The Redis rate limiter mirrors the in-memory one: buckets start full, a
// request claims a token from both, and it refills over time. Time is injected,
// so the behaviour is deterministic.
func TestRedisRateLimitMinuteBurst(t *testing.T) {
	r := redisTestClient(t)
	ctx := context.Background()
	key := "ratelimit:test:minute-burst"
	_ = r.DeleteState(ctx, key)

	now := time.Unix(1_700_000_000, 0)
	for i := 0; i < 5; i++ {
		ok, _, err := r.RateLimitAllow(ctx, key, 5, 1000, now)
		if err != nil {
			t.Fatalf("RateLimitAllow: %v", err)
		}
		if !ok {
			t.Fatalf("request %d should be allowed within the burst", i+1)
		}
	}

	ok, retryAfter, err := r.RateLimitAllow(ctx, key, 5, 1000, now)
	if err != nil {
		t.Fatalf("RateLimitAllow: %v", err)
	}
	if ok {
		t.Fatal("6th request should be denied once the minute burst is spent")
	}
	if retryAfter <= 0 {
		t.Errorf("retry-after = %v, want > 0", retryAfter)
	}

	// After ~12s a token refills (rate 5/60 per second).
	later := now.Add(13 * time.Second)
	if ok, _, _ := r.RateLimitAllow(ctx, key, 5, 1000, later); !ok {
		t.Error("request should be allowed after the bucket refills")
	}
}

// The hourly bucket caps sustained usage independently of the minute bucket.
func TestRedisRateLimitHourCap(t *testing.T) {
	r := redisTestClient(t)
	ctx := context.Background()
	key := "ratelimit:test:hour-cap"
	_ = r.DeleteState(ctx, key)

	now := time.Unix(1_700_000_000, 0)
	for i := 0; i < 3; i++ {
		if ok, _, _ := r.RateLimitAllow(ctx, key, 1000, 3, now); !ok {
			t.Fatalf("request %d should be within the hourly cap", i+1)
		}
	}
	ok, retryAfter, _ := r.RateLimitAllow(ctx, key, 1000, 3, now)
	if ok || retryAfter <= 0 {
		t.Errorf("4th request should be denied by the hour cap (ok=%v, retry=%v)", ok, retryAfter)
	}
}

// A denied request must not consume tokens: one refilled token yields exactly
// one more allowed request.
func TestRedisRateLimitDeniedDoesNotConsume(t *testing.T) {
	r := redisTestClient(t)
	ctx := context.Background()
	key := "ratelimit:test:no-consume"
	_ = r.DeleteState(ctx, key)

	now := time.Unix(1_700_000_000, 0)
	r.RateLimitAllow(ctx, key, 2, 1000, now)
	r.RateLimitAllow(ctx, key, 2, 1000, now)
	for i := 0; i < 3; i++ {
		if ok, _, _ := r.RateLimitAllow(ctx, key, 2, 1000, now); ok {
			t.Fatal("expected denial while bucket empty")
		}
	}
	// Exactly one token refills after 30s (rate 2/60 per second).
	if ok, _, _ := r.RateLimitAllow(ctx, key, 2, 1000, now.Add(30*time.Second)); !ok {
		t.Error("one token should have refilled despite intervening denials")
	}
}
