package database

import (
	"github.com/baely/officetracker/internal/models"
)

type Databaser interface {
	SaveEntry(e models.Entry) error
	GetEntries(userID string, month, year int) (models.Entry, error)
	GetAllEntries(userID string) ([]models.Entry, error)
	GetEntriesForBankYear(userID string, bankYear int) ([]models.Entry, error)
}
