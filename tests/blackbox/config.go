package main

import (
	"github.com/baely/officetracker/internal/config"
)

type Config struct {
	Host string          `envconfig:"HOST"`
	DB   config.Postgres `envconfig:"DB"`
}

func LoadConfig() (Config, error) {
	//var cfg Config
	//err := envconfig.Process("", &cfg)
	//if err != nil {
	//	return Config{}, err
	//}
	//return cfg, nil

	return Config{
		Host: "http://localhost:8080",
		DB: config.Postgres{
			Host:     "localhost",
			Port:     "5432",
			User:     "postgres",
			Password: "postgres",
			DBName:   "postgres",
		},
	}, nil
}
