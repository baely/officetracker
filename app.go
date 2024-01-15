package main

import (
	"os"

	"github.com/baely/officetracker/internal/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	s := server.NewServer(port)
	if err := s.Run(); err != nil {
		panic(err)
	}
}
