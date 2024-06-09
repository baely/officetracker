package database

import (
	"fmt"

	"github.com/baely/officetracker/internal/models"
)

var (
	ErrNoUser = fmt.Errorf("no user found")
)

type Databaser interface {
	SaveEntry(e models.Entry) error
	GetEntries(userID string, month, year int) (models.Entry, error)
	GetAllEntries(userID string) ([]models.Entry, error)
	GetEntriesForBankYear(userID string, bankYear int) ([]models.Entry, error)
	GetUserByGHID(ghID string) (int, error)
	GetUserBySecret(secret string) (int, error)
	GetUser(userID string) (int, error)
	SaveUser(ghID string) (int, error)
}
