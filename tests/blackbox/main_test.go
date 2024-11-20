package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/baely/officetracker/internal/database"
)

func Test(t *testing.T) {
	cfg, err := LoadConfig()
	require.NoError(t, err)

	db, err := database.NewPostgres(cfg.DB)
	require.NoError(t, err)

	Suite{
		cfg: cfg,
		db:  db,
	}.Run(t)
}
