package database

import (
	"fmt"
	"time"

	"github.com/baely/officetracker/pkg/model"
)

var (
	ErrNoUser = fmt.Errorf("no user found")
)

type TokenMetadata struct {
	TokenID   int
	Name      string
	CreatedAt time.Time
	Active    bool
}

type Databaser interface {
	SaveDay(userID int, day int, month int, year int, state model.DayState) error
	GetDay(userID int, day int, month int, year int) (model.DayState, error)
	SaveMonth(userID int, month int, year int, state model.MonthState) error
	GetMonth(userID int, month int, year int) (model.MonthState, error)
	GetYear(userID int, year int, startMonth int) (model.YearState, error)
	SaveNote(userID int, month int, year int, note string) error
	GetNote(userID int, month int, year int) (model.Note, error)
	GetNotes(userID int, year int, startMonth int) (map[int]model.Note, error)

	GetUserByGHID(ghID string) (int, error)
	GetUserBySecret(secret string) (int, error)
	GetUserLinkedAccounts(userID int) ([]model.LinkedAccount, error)

	GetUserByAuth0Sub(sub string) (int, error)
	SaveUserByAuth0Sub(sub string, profile string) (int, error)
	UpdateAuth0Profile(sub string, profile string) error
	LinkAuth0Account(userID int, sub string, profile string) error

	GetThemePreferences(userID int) (model.ThemePreferences, error)
	SaveThemePreferences(userID int, prefs model.ThemePreferences) error
	GetSchedulePreferences(userID int) (model.SchedulePreferences, error)
	SaveSchedulePreferences(userID int, prefs model.SchedulePreferences) error
	GetCalendarPreferences(userID int) (model.CalendarPreferences, error)
	SaveCalendarPreferences(userID int, prefs model.CalendarPreferences) error
	GetTargetPreferences(userID int) (model.TargetPreferences, error)
	SaveTargetPreferences(userID int, prefs model.TargetPreferences) error

	SaveSecret(userID int, secret string, name string) error
	ListActiveTokens(userID int) ([]TokenMetadata, error)
	RevokeToken(userID int, tokenID int) error
	// RevokeSecretByValue deactivates the secret with the given value.
	RevokeSecretByValue(secret string) error

	IsUserSuspended(userID int) (bool, error)

	// Stats dashboard snapshots.
	SaveStatsSnapshot(widgets []model.StatWidget) error
	GetLatestStatsSnapshot() ([]model.StatWidget, time.Time, error)

	// Aggregate stats used by stats collectors. These return
	// non-user-identifiable aggregates for the public dashboard.
	CountTrackedDays() (int, error)
	CountEntriesByState() (map[model.State]int, error)
}
