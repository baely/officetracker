// Command statscollector runs a single stats collection pass and persists a
// snapshot, then exits. It is designed to run as a scheduled Cloud Run Job
// (once per day via Cloud Scheduler).
//
// Build for the full pipeline (including BigQuery-backed usage/cost widgets):
//
//	go build -tags bigquery -o statscollector ./cmd/statscollector
//
// Without the bigquery tag it still runs, emitting the DB-backed and fixed-cost
// widgets only.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/stats"
	"github.com/baely/officetracker/internal/util"
)

func main() {
	util.LoadEnv()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.LoadIntegratedApp()
	if err != nil {
		slog.Error("failed to load config", "error", err.Error())
		os.Exit(1)
	}

	db, err := database.NewPostgres(cfg.Postgres)
	if err != nil {
		slog.Error("failed to connect to database", "error", err.Error())
		os.Exit(1)
	}

	statsCfg, err := stats.LoadConfig()
	if err != nil {
		slog.Error("failed to load stats config", "error", err.Error())
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if _, err := stats.Run(ctx, statsCfg, db); err != nil {
		slog.Error("stats collection failed", "error", err.Error())
		os.Exit(1)
	}

	slog.Info("stats collection complete")
}
