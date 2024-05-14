//go:build standalone

package main

import (
	"flag"

	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/server"
)

func main() {
	port := flag.String("port", "8080", "port to run the server on")
	dbLoc := flag.String("database", "sqlite", "database to use")
	flag.Parse()

	flag.Parse()

	db, err := database.NewSQLiteClient()
	if err != nil {
		panic(err)
	}

	s, err := server.NewServer(cfg, db)
	if err != nil {
		panic(err)
	}
	if err := s.Run(); err != nil {
		panic(err)
	}
}
