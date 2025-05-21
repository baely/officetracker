package v1

import (
	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/report"
)

type Service struct {
	db       database.Databaser
	reporter report.Reporter
	appCfg   config.AppConfigurer // Added appCfg
}

func New(db database.Databaser, reporter report.Reporter, appCfg config.AppConfigurer) *Service { // Added appCfg parameter
	return &Service{
		db:       db,
		reporter: reporter,
		appCfg:   appCfg, // Store appCfg
	}
}
