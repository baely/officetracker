package v1

import (
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/report"
	"github.com/baely/officetracker/pkg/model"
)

type implementation struct {
	db       database.Databaser
	reporter report.Reporter
}

func New(db database.Databaser, reporter report.Reporter) model.Service {
	return &implementation{
		db:       db,
		reporter: reporter,
	}
}
