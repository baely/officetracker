//go:build integrated

// Package main OfficeTracker API
//
//	@title			OfficeTracker API
//	@version		1.0
//	@description	API for tracking Return-to-Office (RTO) compliance and managing attendance records
//	@termsOfService	https://officetracker.app/tos
//
//	@contact.name	OfficeTracker Support
//	@contact.url	https://github.com/baely/officetracker
//	@contact.email	support@officetracker.app
//
//	@license.name	MIT
//	@license.url	https://github.com/baely/officetracker/blob/main/LICENSE
//
//	@host		localhost:8080
//	@BasePath	/api/v1
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				JWT Bearer token for API authentication. Format: Bearer {token}
//
//	@securityDefinitions.apikey	CookieAuth
//	@in							cookie
//	@name						auth-token
//	@description				Session cookie for web authentication
//
//	@schemes	https http
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

	redis, err := database.NewRedis(cfg.Redis)
	if err != nil {
		panic(err)
	}

	reporter := report.New(db)

	s, err := server.NewServer(cfg, db, redis, reporter)
	if err != nil {
		panic(err)
	}
	if err := s.Run(); err != nil {
		panic(err)
	}
}
