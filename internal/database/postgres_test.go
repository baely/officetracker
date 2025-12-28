package database

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/pkg/model"
)

func setupPostgresDB(t *testing.T) (Databaser, func()) {
	ctx := context.Background()

	postgresContainer, err := postgrescontainer.Run(ctx,
		"postgres:16-alpine",
		postgrescontainer.WithDatabase("testdb"),
		postgrescontainer.WithUsername("testuser"),
		postgrescontainer.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2)),
	)
	require.NoError(t, err)

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Run migrations
	sqlDB, err := sql.Open("postgres", connStr)
	require.NoError(t, err)

	// Execute migrations individually
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			user_id SERIAL PRIMARY KEY,
			suspended BOOLEAN NOT NULL DEFAULT false
		)`,
		`CREATE TABLE IF NOT EXISTS entries (
			user_id INTEGER NOT NULL,
			day INTEGER NOT NULL,
			month INTEGER NOT NULL,
			year INTEGER NOT NULL,
			state INTEGER NOT NULL,
			PRIMARY KEY (user_id, day, month, year)
		)`,
		`CREATE TABLE IF NOT EXISTS notes (
			user_id INTEGER NOT NULL,
			month INTEGER NOT NULL,
			year INTEGER NOT NULL,
			notes TEXT NOT NULL,
			PRIMARY KEY (user_id, month, year)
		)`,
		`CREATE TABLE IF NOT EXISTS secrets (
			user_id INTEGER NOT NULL,
			secret TEXT NOT NULL,
			active BOOLEAN NOT NULL DEFAULT true,
			PRIMARY KEY (user_id, secret)
		)`,
		`CREATE TABLE IF NOT EXISTS gh_users (
			gh_id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL UNIQUE
		)`,
		`CREATE TABLE IF NOT EXISTS auth0_users (
			auth0_sub TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL UNIQUE,
			profile TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS user_preferences (
			user_id INTEGER PRIMARY KEY,
			theme TEXT,
			weather_enabled BOOLEAN,
			time_based_enabled BOOLEAN,
			location TEXT,
			schedule_monday_state INTEGER,
			schedule_tuesday_state INTEGER,
			schedule_wednesday_state INTEGER,
			schedule_thursday_state INTEGER,
			schedule_friday_state INTEGER,
			schedule_saturday_state INTEGER,
			schedule_sunday_state INTEGER
		)`,
	}

	for _, migration := range migrations {
		_, err = sqlDB.Exec(migration)
		require.NoError(t, err)
	}

	// Close migration connection before creating new connection
	sqlDB.Close()

	// Extract connection details from connection string
	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err)

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	cfg := config.Postgres{
		Host:     host,
		Port:     port.Port(),
		DBName:   "testdb",
		User:     "testuser",
		Password: "testpass",
	}

	db, err := NewPostgres(cfg)
	require.NoError(t, err)

	cleanup := func() {
		postgresContainer.Terminate(ctx)
	}

	return db, cleanup
}

func TestPostgres_SaveAndGetDay(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

	// Save a day
	err := db.SaveDay(1, 15, 10, 2024, model.DayState{State: model.StateWorkFromHome})
	require.NoError(t, err)

	// Get the day back
	dayState, err := db.GetDay(1, 15, 10, 2024)
	require.NoError(t, err)
	assert.Equal(t, model.StateWorkFromHome, dayState.State)
}

func TestPostgres_GetDay_NotFound(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

	// Get non-existent day
	dayState, err := db.GetDay(1, 15, 10, 2024)
	require.NoError(t, err)
	assert.Equal(t, model.State(0), dayState.State)
}

func TestPostgres_SaveDay_Upsert(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

	// Save initial state
	err := db.SaveDay(1, 15, 10, 2024, model.DayState{State: model.StateWorkFromHome})
	require.NoError(t, err)

	// Upsert with new state
	err = db.SaveDay(1, 15, 10, 2024, model.DayState{State: model.StateWorkFromOffice})
	require.NoError(t, err)

	// Verify new state
	dayState, err := db.GetDay(1, 15, 10, 2024)
	require.NoError(t, err)
	assert.Equal(t, model.StateWorkFromOffice, dayState.State)
}

func TestPostgres_SaveAndGetMonth(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

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

func TestPostgres_GetMonth_Empty(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

	// Get non-existent month
	monthState, err := db.GetMonth(1, 10, 2024)
	require.NoError(t, err)
	assert.Empty(t, monthState.Days)
}

func TestPostgres_GetYear(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

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

func TestPostgres_SaveAndGetNote(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

	// Save note
	err := db.SaveNote(1, 10, 2024, "Test note for October")
	require.NoError(t, err)

	// Get note back
	note, err := db.GetNote(1, 10, 2024)
	require.NoError(t, err)
	assert.Equal(t, "Test note for October", note.Note)
}

func TestPostgres_GetNote_NotFound(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

	// Get non-existent note
	note, err := db.GetNote(1, 10, 2024)
	require.NoError(t, err)
	assert.Empty(t, note.Note)
}

func TestPostgres_SaveNote_Upsert(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

	// Save initial note
	err := db.SaveNote(1, 10, 2024, "First note")
	require.NoError(t, err)

	// Upsert with new note
	err = db.SaveNote(1, 10, 2024, "Updated note")
	require.NoError(t, err)

	// Verify new note
	note, err := db.GetNote(1, 10, 2024)
	require.NoError(t, err)
	assert.Equal(t, "Updated note", note.Note)
}

func TestPostgres_GetNotes(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

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

func TestPostgres_GetNotes_Empty(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

	// Get notes for year with no data
	notes, err := db.GetNotes(1, 2024)
	require.NoError(t, err)
	assert.Empty(t, notes)
}

func TestPostgres_ThemePreferences(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

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

func TestPostgres_SchedulePreferences(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

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

func TestPostgres_SaveSecret_And_GetUserBySecret(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

	// Create a user first
	pgDB := db.(*postgres)
	var userID int
	err := pgDB.db.QueryRow("INSERT INTO users (suspended) VALUES (false) RETURNING user_id").Scan(&userID)
	require.NoError(t, err)

	// Save secret
	err = db.SaveSecret(userID, fmt.Sprintf("test-secret-%d", userID))
	require.NoError(t, err)

	// Get user by secret
	retrievedUserID, err := db.GetUserBySecret(fmt.Sprintf("test-secret-%d", userID))
	require.NoError(t, err)
	assert.Equal(t, userID, retrievedUserID)
}

func TestPostgres_GetUserBySecret_NotFound(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

	// Try to get user with non-existent secret
	_, err := db.GetUserBySecret("non-existent-secret")
	assert.Error(t, err)
	assert.Equal(t, ErrNoUser, err)
}

func TestPostgres_Auth0UserFlow(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

	profile := `{"sub":"github|12345","nickname":"testuser"}`

	// Tier 3: Create new user
	userID, err := db.SaveUserByAuth0Sub("github|12345", profile)
	require.NoError(t, err)
	assert.NotZero(t, userID)

	// Tier 1: Get existing user
	retrievedUserID, err := db.GetUserByAuth0Sub("github|12345")
	require.NoError(t, err)
	assert.Equal(t, userID, retrievedUserID)

	// Update profile
	newProfile := `{"sub":"github|12345","nickname":"updateduser"}`
	err = db.UpdateAuth0Profile("github|12345", newProfile)
	require.NoError(t, err)
}

func TestPostgres_GetUserByAuth0Sub_NotFound(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

	_, err := db.GetUserByAuth0Sub("non-existent-sub")
	assert.Error(t, err)
	assert.Equal(t, ErrNoUser, err)
}

func TestPostgres_IsUserSuspended(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

	// Create a non-suspended user
	pgDB := db.(*postgres)
	var userID int
	err := pgDB.db.QueryRow("INSERT INTO users (suspended) VALUES (false) RETURNING user_id").Scan(&userID)
	require.NoError(t, err)

	suspended, err := db.IsUserSuspended(userID)
	require.NoError(t, err)
	assert.False(t, suspended)

	// Create a suspended user
	var suspendedUserID int
	err2 := pgDB.db.QueryRow("INSERT INTO users (suspended) VALUES (true) RETURNING user_id").Scan(&suspendedUserID)
	require.NoError(t, err2)

	suspended, err = db.IsUserSuspended(suspendedUserID)
	require.NoError(t, err)
	assert.True(t, suspended)
}

func TestPostgres_GetUserByGHID(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

	// Create a user and link GitHub ID
	pgDB := db.(*postgres)
	var userID int
	err := pgDB.db.QueryRow("INSERT INTO users (suspended) VALUES (false) RETURNING user_id").Scan(&userID)
	require.NoError(t, err)

	_, err = pgDB.db.Exec("INSERT INTO gh_users (gh_id, user_id) VALUES ($1, $2)", "12345", userID)
	require.NoError(t, err)

	retrievedUserID, err := db.GetUserByGHID("12345")
	require.NoError(t, err)
	assert.Equal(t, userID, retrievedUserID)
}

func TestPostgres_GetUserByGHID_NotFound(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

	_, err := db.GetUserByGHID("non-existent-gh-id")
	assert.Error(t, err)
	assert.Equal(t, ErrNoUser, err)
}

func TestPostgres_LinkAuth0Account(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

	// Create a user first
	pgDB := db.(*postgres)
	var userID int
	err := pgDB.db.QueryRow("INSERT INTO users (suspended) VALUES (false) RETURNING user_id").Scan(&userID)
	require.NoError(t, err)

	profile := `{"sub":"github|12345","nickname":"testuser"}`

	// Link Auth0 account
	err = db.LinkAuth0Account(userID, "github|12345", profile)
	require.NoError(t, err)

	// Verify the link
	retrievedUserID, err := db.GetUserByAuth0Sub("github|12345")
	require.NoError(t, err)
	assert.Equal(t, userID, retrievedUserID)
}

func TestPostgres_GetUserLinkedAccounts(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

	// Create a user first
	pgDB := db.(*postgres)
	var userID int
	err := pgDB.db.QueryRow("INSERT INTO users (suspended) VALUES (false) RETURNING user_id").Scan(&userID)
	require.NoError(t, err)

	// Link multiple Auth0 accounts
	profile1 := `{"sub":"github|12345","nickname":"user1"}`
	profile2 := `{"sub":"google-oauth2|67890","nickname":"user2"}`

	err = db.LinkAuth0Account(userID, "github|12345", profile1)
	require.NoError(t, err)

	err = db.LinkAuth0Account(userID, "google-oauth2|67890", profile2)
	require.NoError(t, err)

	// Get linked accounts
	accounts, err := db.GetUserLinkedAccounts(userID)
	require.NoError(t, err)
	assert.Len(t, accounts, 2)

	// Verify account details
	accountMap := make(map[string]string)
	for _, acc := range accounts {
		accountMap[acc.Provider] = acc.Nickname
	}

	assert.Equal(t, "user1", accountMap["github"])
	assert.Equal(t, "user2", accountMap["google-oauth2"])
}

func TestPostgres_GetUserLinkedAccounts_Empty(t *testing.T) {
	db, cleanup := setupPostgresDB(t)
	defer cleanup()

	// Create a user without linked accounts
	pgDB := db.(*postgres)
	var userID int
	err := pgDB.db.QueryRow("INSERT INTO users (suspended) VALUES (false) RETURNING user_id").Scan(&userID)
	require.NoError(t, err)

	accounts, err := db.GetUserLinkedAccounts(userID)
	require.NoError(t, err)
	assert.Empty(t, accounts)
}
