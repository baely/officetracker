package v1

import (
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/pkg/model"
)

type implementation struct {
	db database.Databaser
}

func New(db database.Databaser) model.Service {
	return &implementation{
		db: db,
	}
}
