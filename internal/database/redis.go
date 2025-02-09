package database

import (
	"context"
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

func (r *Redis) SetState(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.rdb.Set(ctx, key, value, expiration).Err()
}

func (r *Redis) GetStateInt(ctx context.Context, key string) (int, error) {
	return r.rdb.Get(ctx, key).Int()
}

func (r *Redis) DeleteState(ctx context.Context, key string) error {
	return r.rdb.Del(ctx, key).Err()
}
