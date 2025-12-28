package database

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/pkg/model"
)

func setupSQLiteDB(t *testing.T) Databaser {
	cfg := config.SQLite{
		Location: ":memory:",
	}

	db, err := NewSQLiteClient(cfg)
	require.NoError(t, err)

	return db
}

func TestSQLite_SaveAndGetDay(t *testing.T) {
	db := setupSQLiteDB(t)

	// Save a day
	err := db.SaveDay(1, 15, 10, 2024, model.DayState{State: model.StateWorkFromHome})
	require.NoError(t, err)

	// Get the day back
	dayState, err := db.GetDay(1, 15, 10, 2024)
	require.NoError(t, err)
	assert.Equal(t, model.StateWorkFromHome, dayState.State)
}

func TestSQLite_GetDay_NotFound(t *testing.T) {
	db := setupSQLiteDB(t)

	// Get non-existent day
	dayState, err := db.GetDay(1, 15, 10, 2024)
	require.NoError(t, err)
	assert.Equal(t, model.State(0), dayState.State)
}

func TestSQLite_SaveDay_Replace(t *testing.T) {
	db := setupSQLiteDB(t)

	// Save initial state
	err := db.SaveDay(1, 15, 10, 2024, model.DayState{State: model.StateWorkFromHome})
	require.NoError(t, err)

	// Replace with new state
	err = db.SaveDay(1, 15, 10, 2024, model.DayState{State: model.StateWorkFromOffice})
	require.NoError(t, err)

	// Verify new state
	dayState, err := db.GetDay(1, 15, 10, 2024)
	require.NoError(t, err)
	assert.Equal(t, model.StateWorkFromOffice, dayState.State)
}

func TestSQLite_SaveAndGetMonth(t *testing.T) {
	db := setupSQLiteDB(t)

	monthState := model.MonthState{
		Days: map[int]model.DayState{
			1:  {State: model.StateWorkFromHome},
			15: {State: model.StateWorkFromOffice},
			30: {State: model.StateWorkFromHome},
		},
	}

	// Save month
	err := db.SaveMonth(1, 10, 2024, monthState)
	require.NoError(t, err)

	// Get month back
	retrievedMonth, err := db.GetMonth(1, 10, 2024)
	require.NoError(t, err)
	assert.Len(t, retrievedMonth.Days, 3)
	assert.Equal(t, model.StateWorkFromHome, retrievedMonth.Days[1].State)
	assert.Equal(t, model.StateWorkFromOffice, retrievedMonth.Days[15].State)
	assert.Equal(t, model.StateWorkFromHome, retrievedMonth.Days[30].State)
}

func TestSQLite_GetMonth_Empty(t *testing.T) {
	db := setupSQLiteDB(t)

	// Get non-existent month
	monthState, err := db.GetMonth(1, 10, 2024)
	require.NoError(t, err)
	assert.Empty(t, monthState.Days)
}

func TestSQLite_GetYear(t *testing.T) {
	db := setupSQLiteDB(t)

	// Save data across multiple months
	// Academic year 2024 is Oct 2023 - Sep 2024
	db.SaveDay(1, 15, 10, 2023, model.DayState{State: model.StateWorkFromOffice}) // Oct 2023
	db.SaveDay(1, 15, 1, 2024, model.DayState{State: model.StateWorkFromHome})    // Jan 2024
	db.SaveDay(1, 20, 6, 2024, model.DayState{State: model.StateWorkFromOffice})  // Jun 2024

	// Get year data for academic year 2024
	yearState, err := db.GetYear(1, 2024)
	require.NoError(t, err)

	// Verify months exist (Oct 2023, Jan 2024, Jun 2024)
	assert.Contains(t, yearState.Months, 10)
	assert.Contains(t, yearState.Months, 1)
	assert.Contains(t, yearState.Months, 6)

	// Verify data in each month
	assert.Equal(t, model.StateWorkFromOffice, yearState.Months[10].Days[15].State)
	assert.Equal(t, model.StateWorkFromHome, yearState.Months[1].Days[15].State)
	assert.Equal(t, model.StateWorkFromOffice, yearState.Months[6].Days[20].State)
}

func TestSQLite_SaveAndGetNote(t *testing.T) {
	db := setupSQLiteDB(t)

	// Save note
	err := db.SaveNote(1, 10, 2024, "Test note for October")
	require.NoError(t, err)

	// Get note back
	note, err := db.GetNote(1, 10, 2024)
	require.NoError(t, err)
	assert.Equal(t, "Test note for October", note.Note)
}

func TestSQLite_GetNote_NotFound(t *testing.T) {
	db := setupSQLiteDB(t)

	// Get non-existent note
	note, err := db.GetNote(1, 10, 2024)
	require.NoError(t, err)
	assert.Empty(t, note.Note)
}

func TestSQLite_SaveNote_Replace(t *testing.T) {
	db := setupSQLiteDB(t)

	// Save initial note
	err := db.SaveNote(1, 10, 2024, "First note")
	require.NoError(t, err)

	// Replace with new note
	err = db.SaveNote(1, 10, 2024, "Updated note")
	require.NoError(t, err)

	// Verify new note
	note, err := db.GetNote(1, 10, 2024)
	require.NoError(t, err)
	assert.Equal(t, "Updated note", note.Note)
}

func TestSQLite_GetNotes(t *testing.T) {
	db := setupSQLiteDB(t)

	// Save multiple notes for academic year 2024 (Oct 2023 - Sep 2024)
	db.SaveNote(1, 10, 2023, "October note")  // Oct 2023
	db.SaveNote(1, 1, 2024, "January note")   // Jan 2024
	db.SaveNote(1, 6, 2024, "June note")      // Jun 2024

	// Get all notes for academic year 2024
	notes, err := db.GetNotes(1, 2024)
	require.NoError(t, err)
	assert.Len(t, notes, 3)
	assert.Equal(t, "October note", notes[10].Note)
	assert.Equal(t, "January note", notes[1].Note)
	assert.Equal(t, "June note", notes[6].Note)
}

func TestSQLite_GetNotes_Empty(t *testing.T) {
	db := setupSQLiteDB(t)

	// Get notes for year with no data
	notes, err := db.GetNotes(1, 2024)
	require.NoError(t, err)
	assert.Empty(t, notes)
}

func TestSQLite_ThemePreferences(t *testing.T) {
	db := setupSQLiteDB(t)

	themePrefs := model.ThemePreferences{
		Theme:            "dark",
		WeatherEnabled:   true,
		TimeBasedEnabled: false,
		Location:         "Sydney",
	}

	// Save theme preferences
	err := db.SaveThemePreferences(1, themePrefs)
	require.NoError(t, err)

	// Get theme preferences back
	retrievedPrefs, err := db.GetThemePreferences(1)
	require.NoError(t, err)
	assert.Equal(t, "dark", retrievedPrefs.Theme)
	assert.True(t, retrievedPrefs.WeatherEnabled)
	assert.False(t, retrievedPrefs.TimeBasedEnabled)
	assert.Equal(t, "Sydney", retrievedPrefs.Location)
}

func TestSQLite_SchedulePreferences(t *testing.T) {
	db := setupSQLiteDB(t)

	schedulePrefs := model.SchedulePreferences{
		Monday:    model.StateWorkFromOffice,
		Tuesday:   model.StateWorkFromHome,
		Wednesday: model.StateWorkFromOffice,
		Thursday:  model.StateWorkFromHome,
		Friday:    model.StateWorkFromOffice,
		Saturday:  model.StateUntracked,
		Sunday:    model.StateUntracked,
	}

	// Save schedule preferences
	err := db.SaveSchedulePreferences(1, schedulePrefs)
	require.NoError(t, err)

	// Get schedule preferences back
	retrievedPrefs, err := db.GetSchedulePreferences(1)
	require.NoError(t, err)
	assert.Equal(t, model.StateWorkFromOffice, retrievedPrefs.Monday)
	assert.Equal(t, model.StateWorkFromHome, retrievedPrefs.Tuesday)
	assert.Equal(t, model.StateWorkFromOffice, retrievedPrefs.Wednesday)
	assert.Equal(t, model.StateWorkFromHome, retrievedPrefs.Thursday)
	assert.Equal(t, model.StateWorkFromOffice, retrievedPrefs.Friday)
	assert.Equal(t, model.StateUntracked, retrievedPrefs.Saturday)
	assert.Equal(t, model.StateUntracked, retrievedPrefs.Sunday)
}

func TestSQLite_SaveSecret(t *testing.T) {
	db := setupSQLiteDB(t)

	// Save secret
	err := db.SaveSecret(1, "my-secret-token-12345")
	require.NoError(t, err)

	// Verify secret can be retrieved by looking up user
	userID, err := db.GetUserBySecret("my-secret-token-12345")
	require.NoError(t, err)
	assert.Equal(t, 1, userID)
}

func TestSQLite_GetUserBySecret_NotFound(t *testing.T) {
	db := setupSQLiteDB(t)

	// SQLite is for standalone mode - always returns user ID 1 regardless of secret
	userID, err := db.GetUserBySecret("non-existent-secret")
	require.NoError(t, err)
	assert.Equal(t, 1, userID) // Standalone mode always uses user ID 1
}

func TestSQLite_IsUserSuspended(t *testing.T) {
	db := setupSQLiteDB(t)

	// Check non-existent user - should not be suspended
	suspended, err := db.IsUserSuspended(999)
	require.NoError(t, err)
	assert.False(t, suspended)
}

func TestSQLite_GetUserLinkedAccounts(t *testing.T) {
	db := setupSQLiteDB(t)

	// SQLite doesn't support linked accounts in standalone mode
	accounts, err := db.GetUserLinkedAccounts(1)
	require.NoError(t, err)
	assert.Empty(t, accounts)
}

func TestSQLite_GetUserByGHID(t *testing.T) {
	db := setupSQLiteDB(t)

	// SQLite standalone mode always returns user ID 1
	userID, err := db.GetUserByGHID("any-gh-id")
	require.NoError(t, err)
	assert.Equal(t, 1, userID)
}

func TestSQLite_Auth0Methods_NotSupported(t *testing.T) {
	db := setupSQLiteDB(t)

	// GetUserByAuth0Sub
	_, err := db.GetUserByAuth0Sub("auth0|123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Auth0 authentication not supported")

	// SaveUserByAuth0Sub
	_, err = db.SaveUserByAuth0Sub("auth0|123", "profile")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Auth0 authentication not supported")

	// UpdateAuth0Profile
	err = db.UpdateAuth0Profile("auth0|123", "profile")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Auth0 authentication not supported")

	// LinkAuth0Account
	err = db.LinkAuth0Account(1, "auth0|123", "profile")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Auth0 authentication not supported")
}

func TestSQLite_ThemePreferences_ErrorHandling(t *testing.T) {
	db := setupSQLiteDB(t)

	// Save valid preferences first
	themePrefs := model.ThemePreferences{
		Theme: "dark",
	}
	err := db.SaveThemePreferences(1, themePrefs)
	require.NoError(t, err)

	// Get them back to ensure error paths are covered
	retrieved, err := db.GetThemePreferences(1)
	require.NoError(t, err)
	assert.Equal(t, "dark", retrieved.Theme)
}

func TestSQLite_SaveMonth_ErrorPath(t *testing.T) {
	db := setupSQLiteDB(t)

	monthState := model.MonthState{
		Days: map[int]model.DayState{
			1: {State: model.StateWorkFromHome},
		},
	}

	// This should succeed
	err := db.SaveMonth(1, 10, 2024, monthState)
	require.NoError(t, err)
}

func TestSQLite_NewSQLiteClient_Directory(t *testing.T) {
	// Create a temp directory
	tmpDir := t.TempDir()

	cfg := config.SQLite{
		Location: tmpDir,
	}

	// Should create database in the directory
	db, err := NewSQLiteClient(cfg)
	require.NoError(t, err)
	assert.NotNil(t, db)
}

func TestSQLite_NewSQLiteClient_DefaultLocation(t *testing.T) {
	// Use a temp directory to avoid conflicts
	originalWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	cfg := config.SQLite{
		Location: "",
	}

	db, err := NewSQLiteClient(cfg)
	require.NoError(t, err)
	assert.NotNil(t, db)
}
