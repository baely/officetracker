//go:build integrated

package main

import (
	"log/slog"
	"os"

	"github.com/honeycombio/otel-config-go/otelconfig"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/report"
	"github.com/baely/officetracker/internal/util"

	"github.com/baely/officetracker/internal/server"
)

func main() {
	util.LoadEnv()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	otelShutdown, err := otelconfig.ConfigureOpenTelemetry()
	if err != nil {
		panic(err)
	}
	defer otelShutdown()

	cfg, err := config.LoadIntegratedApp()
	if err != nil {
		panic(err)
	}

	db, err := database.NewPostgres(cfg.Postgres)
	if err != nil {
		panic(err)
	}

	reporter := report.New(db)

	s, err := server.NewServer(cfg, db, reporter)
	if err != nil {
		panic(err)
	}
	if err := s.Run(); err != nil {
		panic(err)
	}
}
