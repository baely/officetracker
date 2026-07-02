package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/baely/officetracker/internal/config"
)

type Redis struct {
	rdb *redis.Client
}

func NewRedis(cfg config.Redis) (*Redis, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Host,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	return &Redis{
		rdb: rdb,
	}, nil
}

// rateLimitScript refills and claims a token from both of a client's token
// buckets in one atomic step. The hash at KEYS[1] holds the minute-bucket
// tokens (m), hour-bucket tokens (h) and the last refill time in microseconds
// (t); buckets start full. ARGV: minute capacity, hour capacity, now in
// microseconds. Returns {allowed, retry-after in milliseconds}.
var rateLimitScript = redis.NewScript(`
local per_minute = tonumber(ARGV[1])
local per_hour = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local state = redis.call('HMGET', KEYS[1], 'm', 'h', 't')
local m = tonumber(state[1])
local h = tonumber(state[2])
local t = tonumber(state[3])
if m == nil then
	m = per_minute
	h = per_hour
	t = now
end

if now > t then
	local elapsed = (now - t) / 1000000
	m = math.min(m + elapsed * per_minute / 60, per_minute)
	h = math.min(h + elapsed * per_hour / 3600, per_hour)
	t = now
end

local allowed = 0
local retry_ms = 0
if m >= 1 and h >= 1 then
	allowed = 1
	m = m - 1
	h = h - 1
else
	local wait_m = 0
	local wait_h = 0
	if m < 1 then wait_m = (1 - m) * 60 / per_minute end
	if h < 1 then wait_h = (1 - h) * 3600 / per_hour end
	retry_ms = math.ceil(math.max(wait_m, wait_h) * 1000)
end

redis.call('HSET', KEYS[1], 'm', m, 'h', h, 't', t)
redis.call('EXPIRE', KEYS[1], 3600)
return {allowed, retry_ms}
`)

// RateLimitAllow atomically claims a token from each of the client's two
// token buckets (a per-minute burst bucket and a per-hour sustained bucket),
// creating them full on first use. It reports whether the request is allowed
// and, when denied, how long the client should wait before retrying. Denied
// requests do not consume tokens. Bucket state expires after an hour idle, by
// which time both buckets are full again anyway.
func (r *Redis) RateLimitAllow(ctx context.Context, key string, perMinute, perHour int, now time.Time) (bool, time.Duration, error) {
	res, err := rateLimitScript.Run(ctx, r.rdb, []string{key}, perMinute, perHour, now.UnixMicro()).Int64Slice()
	if err != nil {
		return false, 0, err
	}
	if len(res) != 2 {
		return false, 0, fmt.Errorf("unexpected rate limit script result: %v", res)
	}
	return res[0] == 1, time.Duration(res[1]) * time.Millisecond, nil
}

func (r *Redis) SetState(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.rdb.Set(ctx, key, value, expiration).Err()
}

func (r *Redis) GetStateInt(ctx context.Context, key string) (int, error) {
	return r.rdb.Get(ctx, key).Int()
}

func (r *Redis) DeleteState(ctx context.Context, key string) error {
	return r.rdb.Del(ctx, key).Err()
}
