package util

import (
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	env := os.Getenv("APP_ENV")
	fn := fmt.Sprintf("%s.env", env)
	p := path.Join("config", fn)
	err := godotenv.Load(p)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to load env: %v", err))
	}
}
