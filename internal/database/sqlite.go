package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"

	_ "github.com/mattn/go-sqlite3"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/pkg/model"
)

const (
	defaultFile = "officetracker.db"
)

type sqliteClient struct {
	cfg    config.SQLite
	db     *sql.DB
	appCfg config.AppConfigurer // To access default theme
}

func NewSQLiteClient(cfg config.SQLite, appCfg config.AppConfigurer) (Databaser, error) {
	db := &sqliteClient{
		cfg:    cfg,
		appCfg: appCfg,
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

func (s *sqliteClient) GetYear(_ int, year int) (model.YearState, error) {
	q := `SELECT Day, Month, State FROM entries WHERE ((Year = ? AND Month > 9) OR (Year = ? AND Month <= 9));`
	rows, err := s.db.Query(q, year-1, year)
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

func (s *sqliteClient) GetNotes(_ int, year int) (map[int]model.Note, error) {
	q := `SELECT Month, Notes FROM notes WHERE ((Year = ? AND Month > 9) OR (Year = ? AND Month <= 9));`
	rows, err := s.db.Query(q, year-1, year)
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

func (s *sqliteClient) GetUser(userID int) (int, string, error) {
	return userID, "", nil
}

func (s *sqliteClient) GetUserGithubAccounts(userID int) ([]string, error) {
	// Standalone mode doesn't use GitHub accounts
	return []string{}, nil
}

func (s *sqliteClient) SaveUserByGHID(ghID string) (int, error) {
	// All users in standalone mode have ID 1
	return 1, nil
}

// SaveUserByGHID's signature updated for interface consistency, though functionality is minimal for SQLite.
func (s *sqliteClient) SaveUserByGHID(ghID string, username string) (int, error) {
	// All users in standalone mode have ID 1
	return 1, nil
}


func (s *sqliteClient) SaveSecret(userID int, secret string) error {
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

func (s *sqliteClient) UpdateUser(_ int, _ string, _ string) error {
	// Standalone mode doesn't use GitHub usernames
	return nil
}

func (s *sqliteClient) UpdateUserGithub(_ int, _ string, _ string) error {
	// Standalone mode doesn't support multiple GitHub accounts
	return nil
}

// GetUserTheme for SQLite. In standalone, might always return default or a fixed theme.
func (s *sqliteClient) GetUserTheme(userID int) (string, error) {
	// Standalone mode typically doesn't have per-user themes.
	// Return the system default theme.
	if s.appCfg != nil {
		return s.appCfg.GetApp().DefaultTheme, nil
	}
	return "default", nil // Fallback if appCfg is somehow nil
}

// SetUserTheme for SQLite. Likely a no-op in standalone mode.
func (s *sqliteClient) SetUserTheme(userID int, theme string) error {
	// Standalone mode typically doesn't store per-user themes.
	// This could potentially write to a local config if needed, but for now, it's a no-op.
	slog.Info("SetUserTheme called in SQLite mode (no-op)", "userID", userID, "theme", theme)
	return nil
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

-- Minimal users table for SQLite to align with interface, though not fully utilized.
-- In standalone, user management is very basic (often just a single implicit user).
CREATE TABLE IF NOT EXISTS users (
    user_id INTEGER PRIMARY KEY AUTOINCREMENT, -- In SQLite, user_id is often fixed or not used.
    theme TEXT DEFAULT 'default' -- Store theme preference if ever needed.
);
`

	if _, err = db.Exec(sqlCreate); err != nil {
		return err
	}
	s.db = db

	// For standalone, ensure a default user record exists if we plan to use SetUserTheme.
	// However, given current SaveUserByGHID returns 1, it implies a fixed user.
	// For simplicity, we'll assume SetUserTheme remains a no-op or operates on a known row if necessary.
	// Example: _, err = db.Exec("INSERT OR IGNORE INTO users (user_id, theme) VALUES (1, ?)", s.appCfg.GetApp().DefaultTheme)

	return nil
}
