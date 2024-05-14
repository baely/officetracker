//go:build integrated
// +build integrated

package main

import (
	"os"

	"github.com/baely/officetracker/internal/database"

	"github.com/baely/officetracker/internal/server"
	"github.com/baely/officetracker/internal/util"
)

func main() {
	util.LoadEnv()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := database.NewFirestoreClient()
	if err != nil {
		panic(err)
	}

	s, err := server.NewServer(port, db)
	if err != nil {
		panic(err)
	}
	if err := s.Run(); err != nil {
		panic(err)
	}
}
