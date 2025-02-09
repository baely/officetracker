package v1

import (
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/report"
)

type Service struct {
	db       database.Databaser
	reporter report.Reporter
}

func New(db database.Databaser, reporter report.Reporter) *Service {
	return &Service{
		db:       db,
		reporter: reporter,
	}
}
