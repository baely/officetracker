package main

import (
	"database/sql"
	"embed"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"slices"

	"github.com/baely/officetracker/internal/database"
)

var (
	//go:embed *.sql
	migration embed.FS
)

var (
	postgresUser = os.Getenv("POSTGRES_USER")
	postgresPass = os.Getenv("POSTGRES_PASSWORD")
	postgresHost = os.Getenv("POSTGRES_HOST")
	postgresPort = os.Getenv("POSTGRES_PORT")
	postgresDB   = os.Getenv("POSTGRES_DB")
)

func main() {
	conn, err := sql.Open("postgres", fmt.Sprintf(database.PqConnFormat, postgresHost, postgresPort, postgresUser, postgresPass, postgresDB))
	defer func(conn *sql.DB) {
		err = conn.Close()
		if err != nil {
			panic(err)
		}
	}(conn)
	if err != nil {
		panic(err)
	}
	var filenames []string
	files, err := migration.ReadDir(".")
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		filenames = append(filenames, file.Name())
	}
	slices.Sort(filenames)
	for _, k := range filenames {
		slog.Info(fmt.Sprintf("Applying migration %s", k))
		f, err := migration.ReadFile(k)
		if err != nil {
			panic(err)
		}
		_, err = conn.Exec(string(f))
		if err != nil {
			panic(err)
		}
	}
}
