//go:build standalone

package main

import (
	"flag"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/server"
)

func main() {
	port := flag.String("port", "8080", "port to run the server on")
	dbLoc := flag.String("database", "", "database to use")
	flag.Parse()

	cfg := config.StandaloneApp{
		App: config.App{
			Port: *port,
		},
		SQLite: config.SQLite{
			Location: *dbLoc,
		},
	}

	db, err := database.NewSQLiteClient(cfg.SQLite)
	if err != nil {
		panic(err)
	}

	s, err := server.NewStandaloneServer(cfg, db)
	if err != nil {
		panic(err)
	}
	if err := s.Run(); err != nil {
		panic(err)
	}
}
