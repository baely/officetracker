package main

import (
	"flag"
	"github.com/baely/officetracker/internal/database"
	"os"

	"github.com/baely/officetracker/internal/server"
	"github.com/baely/officetracker/internal/util"
)

func main() {
	util.LoadEnv()

	standalone := flag.Bool("standalone", false, "run in standalone mode")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	var db database.Databaser
	db := database.NewClient(standalone)

	s, err := server.NewServer(port, db)
	if err != nil {
		panic(err)
	}
	if err := s.Run(); err != nil {
		panic(err)
	}
}
