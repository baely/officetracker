package util

import (
	"os"
)

const (
	DemoUserId = "42069"
)

var (
	demo string
)

func Demo() bool {
	if demo == "" {
		demo = os.Getenv("DEMO")
	}

	return demo == "true"
}
