//go:build standalone

package main

import (
	"flag"
	"os/exec"
	"time"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/server"
)

func main() {
	port := flag.String("port", "8080", "port to run the server on")
	dbLoc := flag.String("database", "officetracker.db", "database to use")
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

	s, err := server.NewServer(cfg, db)
	if err != nil {
		panic(err)
	}

	go func() {
		time.Sleep(1 * time.Second)
		_ = exec.Command("open", "http://localhost:"+cfg.App.Port).Start()
	}()

	if err := s.Run(); err != nil {
		panic(err)
	}
}
