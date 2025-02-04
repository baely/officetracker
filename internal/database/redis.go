package database

import (
	"github.com/redis/go-redis/v9"

	"github.com/baely/officetracker/internal/config"
)

type Redis struct {
	rdb *redis.Client
}

func NewRedis(cfg config.Redis) (Redis, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Host,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	return Redis{
		rdb: rdb,
	}, nil
}
