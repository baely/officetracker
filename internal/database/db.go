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
	GetUser(userID int) (int, string, error)
	GetUserGithubAccounts(userID int) ([]string, error)
	GetUserLinkedAccounts(userID int) ([]model.LinkedAccount, error)
	SaveUserByGHID(ghID string) (int, error)
	UpdateUser(userID int, ghID string, username string) error
	UpdateUserGithub(userID int, ghID string, username string) error

	GetUserByAuth0Sub(sub string) (int, error)
	SaveUserByAuth0Sub(sub string, profile string) (int, error)
	UpdateAuth0Profile(sub string, profile string) error
	LinkAuth0Account(userID int, sub string, profile string) error

	GetThemePreferences(userID int) (model.ThemePreferences, error)
	SaveThemePreferences(userID int, prefs model.ThemePreferences) error
	GetSchedulePreferences(userID int) (model.SchedulePreferences, error)
	SaveSchedulePreferences(userID int, prefs model.SchedulePreferences) error

	SaveSecret(userID int, secret string) error

	IsUserSuspended(userID int) (bool, error)
}
