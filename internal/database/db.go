package database

import (
	"fmt"

	"github.com/baely/officetracker/pkg/model"
)

var (
	ErrNoUser = fmt.Errorf("no user found")
)

type Databaser interface {
	SaveDay(userID int, day int, month int, year int, state model.DayState) error
	GetDay(userID int, day int, month int, year int) (model.DayState, error)
	SaveMonth(userID int, month int, year int, state model.MonthState) error
	GetMonth(userID int, month int, year int) (model.MonthState, error)
	GetYear(userID int, year int) (model.YearState, error)
	SaveNote(userID int, month int, year int, note string) error
	GetNote(userID int, month int, year int) (model.Note, error)
	GetNotes(userID int, year int) (map[int]model.Note, error)

	GetUserByGHID(ghID string) (int, error)
	GetUserBySecret(secret string) (int, error)
	GetUser(userID int) (int, error)
	SaveUserByGHID(ghID string) (int, error)

	SaveSecret(userID int, secret string) error
}
