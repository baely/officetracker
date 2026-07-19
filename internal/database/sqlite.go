package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/util"
	"github.com/baely/officetracker/pkg/model"
)

const (
	defaultFile = "officetracker.db"
)

type sqliteClient struct {
	cfg config.SQLite
	db  *sql.DB
}

func NewSQLiteClient(cfg config.SQLite) (Databaser, error) {
	db := &sqliteClient{
		cfg: cfg,
	}

	if db.cfg.Location == "" {
		db.cfg.Location = defaultFile
	}

	fi, err := os.Stat(db.cfg.Location)
	if os.IsNotExist(err) {
		_, err = os.Create(db.cfg.Location)
		fi, err = os.Stat(db.cfg.Location)
	}
	if err != nil {
		return nil, err
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():
		db.cfg.Location = path.Join(db.cfg.Location, defaultFile)
	case mode.IsRegular():
		// do nothing
	default:
		return nil, errors.New("invalid file")
	}

	if err = db.initConnection(); err != nil {
		return nil, err
	}

	return db, nil
}

func (s *sqliteClient) SaveDay(_ int, day int, month int, year int, state model.DayState) error {
	q := `INSERT OR REPLACE INTO entries (Day, Month, Year, State) VALUES (?, ?, ?, ?);`
	_, err := s.db.Exec(q, day, month, year, state.State)
	return err
}

func (s *sqliteClient) GetDay(_ int, day int, month int, year int) (model.DayState, error) {
	q := `SELECT State FROM entries WHERE Day = ? AND Month = ? AND Year = ?;`
	row := s.db.QueryRow(q, day, month, year)
	var state model.DayState
	err := row.Scan(&state.State)
	if errors.Is(err, sql.ErrNoRows) {
		return model.DayState{}, nil
	}
	return state, err
}

func (s *sqliteClient) SaveMonth(_ int, month int, year int, state model.MonthState) error {
	q := `INSERT OR REPLACE INTO entries (Day, Month, Year, State) VALUES (?, ?, ?, ?);`
	for day, dayState := range state.Days {
		_, err := s.db.Exec(q, day, month, year, dayState.State)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *sqliteClient) GetMonth(_ int, month int, year int) (model.MonthState, error) {
	q := `SELECT Day, State FROM entries WHERE Month = ? AND Year = ?;`
	rows, err := s.db.Query(q, month, year)
	if err != nil {
		return model.MonthState{}, err
	}
	defer rows.Close()
	monthState := model.MonthState{
		Days: make(map[int]model.DayState),
	}
	for rows.Next() {
		var day int
		var state model.DayState
		err = rows.Scan(&day, &state.State)
		if err != nil {
			return model.MonthState{}, err
		}
		monthState.Days[day] = state
	}
	return monthState, nil
}

func (s *sqliteClient) GetYear(_ int, year int, startMonth int) (model.YearState, error) {
	startMonth = util.NormaliseStartMonth(startMonth)
	firstYear, secondYear := util.TrackingYearCalendarYears(year, startMonth)
	q := `SELECT Day, Month, State FROM entries WHERE ((Year = ? AND Month >= ?) OR (Year = ? AND Month < ?));`
	rows, err := s.db.Query(q, firstYear, startMonth, secondYear, startMonth)
	if err != nil {
		return model.YearState{}, err
	}
	defer rows.Close()
	yearState := model.YearState{
		Months: make(map[int]model.MonthState),
	}
	for rows.Next() {
		var month int
		var day int
		var state model.DayState
		err = rows.Scan(&day, &month, &state.State)
		if err != nil {
			return model.YearState{}, err
		}
		if _, ok := yearState.Months[month]; !ok {
			yearState.Months[month] = model.MonthState{
				Days: make(map[int]model.DayState),
			}
		}
		yearState.Months[month].Days[day] = state
	}
	return yearState, nil
}

func (s *sqliteClient) SaveNote(_ int, month int, year int, note string) error {
	q := `INSERT OR REPLACE INTO notes (Month, Year, Notes) VALUES (?, ?, ?);`
	_, err := s.db.Exec(q, month, year, note)
	return err
}

func (s *sqliteClient) GetNote(_ int, month int, year int) (model.Note, error) {
	q := `SELECT Notes FROM notes WHERE Month = ? AND Year = ?;`
	row := s.db.QueryRow(q, month, year)
	var note model.Note
	err := row.Scan(&note.Note)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Note{}, nil
	}
	return note, err
}

func (s *sqliteClient) GetNotes(_ int, year int, startMonth int) (map[int]model.Note, error) {
	startMonth = util.NormaliseStartMonth(startMonth)
	firstYear, secondYear := util.TrackingYearCalendarYears(year, startMonth)
	q := `SELECT Month, Notes FROM notes WHERE ((Year = ? AND Month >= ?) OR (Year = ? AND Month < ?));`
	rows, err := s.db.Query(q, firstYear, startMonth, secondYear, startMonth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	notes := make(map[int]model.Note)
	for rows.Next() {
		var month int
		var note model.Note
		err = rows.Scan(&month, &note.Note)
		if err != nil {
			return nil, err
		}
		notes[month] = note
	}
	return notes, nil
}

func (s *sqliteClient) GetUserLinkedAccounts(userID int) ([]model.LinkedAccount, error) {
	// Standalone mode doesn't use linked accounts
	return []model.LinkedAccount{}, nil
}

func (s *sqliteClient) SaveSecret(userID int, secret string, name string) error {
	// Standalone mode doesn't use secrets
	return nil
}

func (s *sqliteClient) ListActiveTokens(userID int) ([]TokenMetadata, error) {
	// Standalone mode doesn't use secrets
	return []TokenMetadata{}, nil
}

func (s *sqliteClient) RevokeToken(userID int, tokenID int) error {
	// Standalone mode doesn't use secrets
	return nil
}

func (s *sqliteClient) RevokeSecretByValue(secret string) error {
	// Standalone mode doesn't use secrets
	return nil
}

func (s *sqliteClient) GetUserByGHID(_ string) (int, error) {
	// All users in standalone mode have ID 1
	return 1, nil
}

func (s *sqliteClient) GetUserBySecret(_ string) (int, error) {
	// All users in standalone mode have ID 1
	return 1, nil
}

func (s *sqliteClient) GetUserByAuth0Sub(_ string) (int, error) {
	// Auth0 not supported in standalone mode
	return 0, fmt.Errorf("Auth0 authentication not supported in standalone mode")
}

func (s *sqliteClient) SaveUserByAuth0Sub(_ string, _ string) (int, error) {
	// Auth0 not supported in standalone mode
	return 0, fmt.Errorf("Auth0 authentication not supported in standalone mode")
}

func (s *sqliteClient) UpdateAuth0Profile(_ string, _ string) error {
	// Auth0 not supported in standalone mode
	return fmt.Errorf("Auth0 authentication not supported in standalone mode")
}

func (s *sqliteClient) LinkAuth0Account(_ int, _ string, _ string) error {
	// Auth0 not supported in standalone mode
	return fmt.Errorf("Auth0 authentication not supported in standalone mode")
}

func (s *sqliteClient) GetThemePreferences(_ int) (model.ThemePreferences, error) {
	// Check if the preferences table exists
	q := `SELECT name FROM sqlite_master WHERE type='table' AND name='user_preferences';`
	row := s.db.QueryRow(q)
	var tableName string
	err := row.Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		// Table doesn't exist, return defaults
		return model.ThemePreferences{
			Theme:            "default",
			WeatherEnabled:   false,
			TimeBasedEnabled: false,
		}, nil
	}

	// Table exists, get preferences
	q = `SELECT theme, weather_enabled, time_based_enabled, location FROM user_preferences LIMIT 1;`
	row = s.db.QueryRow(q)
	var prefs model.ThemePreferences
	var location sql.NullString

	err = row.Scan(&prefs.Theme, &prefs.WeatherEnabled, &prefs.TimeBasedEnabled, &location)
	if errors.Is(err, sql.ErrNoRows) {
		// No preferences yet, return defaults
		return model.ThemePreferences{
			Theme:            "default",
			WeatherEnabled:   false,
			TimeBasedEnabled: false,
		}, nil
	}

	if err != nil {
		return model.ThemePreferences{}, err
	}

	if location.Valid {
		prefs.Location = location.String
	}

	return prefs, nil
}

func (s *sqliteClient) SaveThemePreferences(_ int, prefs model.ThemePreferences) error {
	// Make sure the table exists
	q := `CREATE TABLE IF NOT EXISTS user_preferences (
        theme TEXT DEFAULT 'default',
        weather_enabled INTEGER DEFAULT 0,
        time_based_enabled INTEGER DEFAULT 0,
        location TEXT DEFAULT NULL
    );`

	_, err := s.db.Exec(q)
	if err != nil {
		return err
	}

	// Check if any preferences exist
	q = `SELECT COUNT(*) FROM user_preferences;`
	row := s.db.QueryRow(q)
	var count int
	err = row.Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		// Insert new preferences
		q = `INSERT INTO user_preferences (theme, weather_enabled, time_based_enabled, location) 
             VALUES (?, ?, ?, ?);`
		_, err = s.db.Exec(q, prefs.Theme, prefs.WeatherEnabled, prefs.TimeBasedEnabled, prefs.Location)
	} else {
		// Update existing preferences
		q = `UPDATE user_preferences SET theme = ?, weather_enabled = ?, time_based_enabled = ?, location = ?;`
		_, err = s.db.Exec(q, prefs.Theme, prefs.WeatherEnabled, prefs.TimeBasedEnabled, prefs.Location)
	}

	return err
}

func (s *sqliteClient) GetSchedulePreferences(_ int) (model.SchedulePreferences, error) {
	// First make sure schedule columns exist
	q := `ALTER TABLE user_preferences ADD COLUMN schedule_monday_state INTEGER DEFAULT 0;`
	s.db.Exec(q) // Ignore error in case column already exists
	q = `ALTER TABLE user_preferences ADD COLUMN schedule_tuesday_state INTEGER DEFAULT 0;`
	s.db.Exec(q)
	q = `ALTER TABLE user_preferences ADD COLUMN schedule_wednesday_state INTEGER DEFAULT 0;`
	s.db.Exec(q)
	q = `ALTER TABLE user_preferences ADD COLUMN schedule_thursday_state INTEGER DEFAULT 0;`
	s.db.Exec(q)
	q = `ALTER TABLE user_preferences ADD COLUMN schedule_friday_state INTEGER DEFAULT 0;`
	s.db.Exec(q)
	q = `ALTER TABLE user_preferences ADD COLUMN schedule_saturday_state INTEGER DEFAULT 0;`
	s.db.Exec(q)
	q = `ALTER TABLE user_preferences ADD COLUMN schedule_sunday_state INTEGER DEFAULT 0;`
	s.db.Exec(q)

	q = `SELECT 
		COALESCE(schedule_monday_state, 0),
		COALESCE(schedule_tuesday_state, 0),
		COALESCE(schedule_wednesday_state, 0),
		COALESCE(schedule_thursday_state, 0),
		COALESCE(schedule_friday_state, 0),
		COALESCE(schedule_saturday_state, 0),
		COALESCE(schedule_sunday_state, 0)
		FROM user_preferences LIMIT 1;`

	row := s.db.QueryRow(q)
	var prefs model.SchedulePreferences
	var monday, tuesday, wednesday, thursday, friday, saturday, sunday int

	err := row.Scan(&monday, &tuesday, &wednesday, &thursday, &friday, &saturday, &sunday)
	if err != nil {
		// Return default values if no preferences exist
		return model.SchedulePreferences{
			Monday:    model.StateUntracked,
			Tuesday:   model.StateUntracked,
			Wednesday: model.StateUntracked,
			Thursday:  model.StateUntracked,
			Friday:    model.StateUntracked,
			Saturday:  model.StateUntracked,
			Sunday:    model.StateUntracked,
		}, nil
	}

	prefs.Monday = model.State(monday)
	prefs.Tuesday = model.State(tuesday)
	prefs.Wednesday = model.State(wednesday)
	prefs.Thursday = model.State(thursday)
	prefs.Friday = model.State(friday)
	prefs.Saturday = model.State(saturday)
	prefs.Sunday = model.State(sunday)

	return prefs, nil
}

func (s *sqliteClient) SaveSchedulePreferences(_ int, prefs model.SchedulePreferences) error {
	// First make sure schedule columns exist
	q := `ALTER TABLE user_preferences ADD COLUMN schedule_monday_state INTEGER DEFAULT 0;`
	s.db.Exec(q) // Ignore error in case column already exists
	q = `ALTER TABLE user_preferences ADD COLUMN schedule_tuesday_state INTEGER DEFAULT 0;`
	s.db.Exec(q)
	q = `ALTER TABLE user_preferences ADD COLUMN schedule_wednesday_state INTEGER DEFAULT 0;`
	s.db.Exec(q)
	q = `ALTER TABLE user_preferences ADD COLUMN schedule_thursday_state INTEGER DEFAULT 0;`
	s.db.Exec(q)
	q = `ALTER TABLE user_preferences ADD COLUMN schedule_friday_state INTEGER DEFAULT 0;`
	s.db.Exec(q)
	q = `ALTER TABLE user_preferences ADD COLUMN schedule_saturday_state INTEGER DEFAULT 0;`
	s.db.Exec(q)
	q = `ALTER TABLE user_preferences ADD COLUMN schedule_sunday_state INTEGER DEFAULT 0;`
	s.db.Exec(q)

	// Check if any preferences exist
	q = `SELECT COUNT(*) FROM user_preferences;`
	row := s.db.QueryRow(q)
	var count int
	err := row.Scan(&count)
	if err != nil {
		return err
	}

	monday := int(prefs.Monday)
	tuesday := int(prefs.Tuesday)
	wednesday := int(prefs.Wednesday)
	thursday := int(prefs.Thursday)
	friday := int(prefs.Friday)
	saturday := int(prefs.Saturday)
	sunday := int(prefs.Sunday)

	if count == 0 {
		// Insert new preferences
		q = `INSERT INTO user_preferences (schedule_monday_state, schedule_tuesday_state, schedule_wednesday_state, 
		     schedule_thursday_state, schedule_friday_state, schedule_saturday_state, schedule_sunday_state) 
             VALUES (?, ?, ?, ?, ?, ?, ?);`
		_, err = s.db.Exec(q, monday, tuesday, wednesday, thursday, friday, saturday, sunday)
	} else {
		// Update existing preferences
		q = `UPDATE user_preferences SET schedule_monday_state = ?, schedule_tuesday_state = ?, schedule_wednesday_state = ?,
		     schedule_thursday_state = ?, schedule_friday_state = ?, schedule_saturday_state = ?, schedule_sunday_state = ?;`
		_, err = s.db.Exec(q, monday, tuesday, wednesday, thursday, friday, saturday, sunday)
	}

	return err
}

func (s *sqliteClient) GetCalendarPreferences(_ int) (model.CalendarPreferences, error) {
	prefs := model.CalendarPreferences{TrackingYearStartMonth: model.DefaultTrackingYearStartMonth}

	// Check if the preferences table exists; if not, return defaults.
	q := `SELECT name FROM sqlite_master WHERE type='table' AND name='user_preferences';`
	var tableName string
	if err := s.db.QueryRow(q).Scan(&tableName); errors.Is(err, sql.ErrNoRows) {
		return prefs, nil
	}

	// Make sure the column exists (ignore error if it already does).
	s.db.Exec(`ALTER TABLE user_preferences ADD COLUMN tracking_year_start_month INTEGER DEFAULT 10;`)

	q = `SELECT COALESCE(tracking_year_start_month, 10) FROM user_preferences LIMIT 1;`
	var startMonth int
	if err := s.db.QueryRow(q).Scan(&startMonth); err != nil {
		// No row yet (or other read issue) - fall back to defaults.
		return prefs, nil
	}

	prefs.TrackingYearStartMonth = util.NormaliseStartMonth(startMonth)
	return prefs, nil
}

func (s *sqliteClient) SaveCalendarPreferences(_ int, prefs model.CalendarPreferences) error {
	startMonth := util.NormaliseStartMonth(prefs.TrackingYearStartMonth)

	// Make sure the table and column exist.
	q := `CREATE TABLE IF NOT EXISTS user_preferences (
        theme TEXT DEFAULT 'default',
        weather_enabled INTEGER DEFAULT 0,
        time_based_enabled INTEGER DEFAULT 0,
        location TEXT DEFAULT NULL
    );`
	if _, err := s.db.Exec(q); err != nil {
		return err
	}
	s.db.Exec(`ALTER TABLE user_preferences ADD COLUMN tracking_year_start_month INTEGER DEFAULT 10;`)

	q = `SELECT COUNT(*) FROM user_preferences;`
	var count int
	if err := s.db.QueryRow(q).Scan(&count); err != nil {
		return err
	}

	if count == 0 {
		_, err := s.db.Exec(`INSERT INTO user_preferences (tracking_year_start_month) VALUES (?);`, startMonth)
		return err
	}
	_, err := s.db.Exec(`UPDATE user_preferences SET tracking_year_start_month = ?;`, startMonth)
	return err
}

func (s *sqliteClient) GetTargetPreferences(_ int) (model.TargetPreferences, error) {
	var prefs model.TargetPreferences

	// Check if the preferences table exists; if not, return defaults.
	q := `SELECT name FROM sqlite_master WHERE type='table' AND name='user_preferences';`
	var tableName string
	if err := s.db.QueryRow(q).Scan(&tableName); errors.Is(err, sql.ErrNoRows) {
		return prefs, nil
	}

	// Make sure the column exists (ignore error if it already does).
	s.db.Exec(`ALTER TABLE user_preferences ADD COLUMN target_percent INTEGER DEFAULT 0;`)

	q = `SELECT COALESCE(target_percent, 0) FROM user_preferences LIMIT 1;`
	var target int
	if err := s.db.QueryRow(q).Scan(&target); err != nil {
		// No row yet (or other read issue) - fall back to defaults.
		return prefs, nil
	}

	prefs.TargetPercent = util.ClampTargetPercent(target)
	return prefs, nil
}

func (s *sqliteClient) SaveTargetPreferences(_ int, prefs model.TargetPreferences) error {
	target := util.ClampTargetPercent(prefs.TargetPercent)

	// Make sure the table and column exist.
	q := `CREATE TABLE IF NOT EXISTS user_preferences (
        theme TEXT DEFAULT 'default',
        weather_enabled INTEGER DEFAULT 0,
        time_based_enabled INTEGER DEFAULT 0,
        location TEXT DEFAULT NULL
    );`
	if _, err := s.db.Exec(q); err != nil {
		return err
	}
	s.db.Exec(`ALTER TABLE user_preferences ADD COLUMN target_percent INTEGER DEFAULT 0;`)

	q = `SELECT COUNT(*) FROM user_preferences;`
	var count int
	if err := s.db.QueryRow(q).Scan(&count); err != nil {
		return err
	}

	if count == 0 {
		_, err := s.db.Exec(`INSERT INTO user_preferences (target_percent) VALUES (?);`, target)
		return err
	}
	_, err := s.db.Exec(`UPDATE user_preferences SET target_percent = ?;`, target)
	return err
}

func (s *sqliteClient) IsUserSuspended(_ int) (bool, error) {
	// Standalone mode doesn't support suspension
	return false, nil
}

// Stats snapshots are only used by the integrated deployment's collector job.
// Standalone mode has no public dashboard, so these are no-ops.
func (s *sqliteClient) SaveStatsSnapshot(_ []model.StatWidget) error {
	return nil
}

func (s *sqliteClient) GetLatestStatsSnapshot() ([]model.StatWidget, time.Time, error) {
	return nil, time.Time{}, nil
}

func (s *sqliteClient) CountTrackedDays() (int, error) {
	q := `SELECT COUNT(*) FROM entries WHERE State != ?;`
	var count int
	err := s.db.QueryRow(q, int(model.StateUntracked)).Scan(&count)
	return count, err
}

func (s *sqliteClient) CountEntriesByState() (map[model.State]int, error) {
	q := `SELECT State, COUNT(*) FROM entries WHERE State != ? GROUP BY State;`
	result := make(map[model.State]int)
	rows, err := s.db.Query(q, int(model.StateUntracked))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var state, count int
		if err := rows.Scan(&state, &count); err != nil {
			return nil, err
		}
		result[model.State(state)] = count
	}
	return result, rows.Err()
}

func (s *sqliteClient) initConnection() error {
	slog.Info(fmt.Sprintf("Connecting to sqlite database: %s", s.cfg.Location))
	db, err := sql.Open("sqlite3", s.cfg.Location)
	if err != nil {
		return err
	}

	sqlCreate := `CREATE TABLE IF NOT EXISTS entries (
    Day INTEGER,
    Month INTEGER,
    Year INTEGER,
    State INTEGER,
    PRIMARY KEY (Day, Month, Year)
);

CREATE TABLE IF NOT EXISTS notes (
    Month INTEGER,
    Year INTEGER,
    Notes TEXT,
    PRIMARY KEY (Month, Year)
);

CREATE TABLE IF NOT EXISTS user_preferences (
    theme TEXT DEFAULT 'default',
    weather_enabled INTEGER DEFAULT 0,
    time_based_enabled INTEGER DEFAULT 0,
    location TEXT DEFAULT NULL
);`

	if _, err = db.Exec(sqlCreate); err != nil {
		return err
	}
	s.db = db

	return nil
}
