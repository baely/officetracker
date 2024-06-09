package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strconv"

	_ "github.com/mattn/go-sqlite3"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/models"
)

const (
	defaultFile = "officetracker.db"
)

type sqliteClient struct {
	cfg config.SQLite
	db  *sql.DB
}

type entry struct {
	UserID, Day, Month, Year, State int
}

type note struct {
	UserID, Month, Year int
	Notes               string
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

func (s *sqliteClient) SaveEntry(e models.Entry) error {
	entries, notes, err := mapFromModel(e)
	if err != nil {
		return err
	}

	for _, e2 := range entries {
		if err = s.insertEntry(e2); err != nil {
			return err
		}
	}

	if err = s.insertNote(notes); err != nil {
		return err
	}

	return nil
}

func (s *sqliteClient) GetEntries(_ string, month, year int) (models.Entry, error) {
	entries, err := s.getEntriesForMonth(month, year)
	if err != nil {
		return models.Entry{}, err
	}

	notes, err := s.getNotesForMonth(month, year)
	if err != nil {
		return models.Entry{}, err
	}

	return mapToModel(entries, notes), nil
}

func (s *sqliteClient) GetAllEntries(_ string) ([]models.Entry, error) {
	m := make(map[string][]entry)

	entries, err := s.getAllEntries()
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		k := fmt.Sprintf("%d-%d", e.Month, e.Year)
		m[k] = append(m[k], e)
	}

	var res []models.Entry
	for _, v := range m {
		notes, err := s.getNotesForMonth(v[0].Month, v[0].Year)
		if err != nil {
			return nil, err
		}
		res = append(res, mapToModel(v, notes))
	}

	return res, nil
}

func (s *sqliteClient) GetEntriesForBankYear(_ string, bankYear int) ([]models.Entry, error) {
	startMonth, startYear := 10, bankYear-1
	endMonth, endYear := 9, bankYear

	var res []models.Entry

	for month := startMonth; month <= 12; month++ {
		e, err := s.GetEntries("", month, startYear)
		if err != nil {
			return nil, err
		}
		res = append(res, e)
	}

	for month := 1; month <= endMonth; month++ {
		e, err := s.GetEntries("", month, endYear)
		if err != nil {
			return nil, err
		}
		res = append(res, e)
	}

	return res, nil
}

func (s *sqliteClient) GetUserByGHID(_ string) (int, error) {
	return 42069, nil
}

func (s *sqliteClient) GetUser(_ string) (int, error) {
	return 42069, nil
}

func (s *sqliteClient) SaveUser(_ string) (int, error) {
	return 42069, nil
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
)`

	if _, err = db.Exec(sqlCreate); err != nil {
		return err
	}
	s.db = db

	return nil
}

func (s *sqliteClient) insertEntry(e entry) error {
	q := `INSERT OR REPLACE INTO entries (Day, Month, Year, State) VALUES (?, ?, ?, ?);`
	_, err := s.db.Exec(q, e.Day, e.Month, e.Year, e.State)
	return err
}

func (s *sqliteClient) insertNote(n note) error {
	q := `INSERT OR REPLACE INTO notes (Month, Year, Notes) VALUES (?, ?, ?);`
	_, err := s.db.Exec(q, n.Month, n.Year, n.Notes)
	return err
}

func (s *sqliteClient) getEntriesForMonth(month, year int) ([]entry, error) {
	q := `SELECT * FROM entries WHERE Month = ? AND Year = ?;`
	rows, err := s.db.Query(q, month, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []entry

	for rows.Next() {

		var e entry
		e, err = mapEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	return entries, nil
}

func (s *sqliteClient) getAllEntries() ([]entry, error) {
	q := `SELECT * FROM entries;`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []entry
	for rows.Next() {
		var e entry
		e, err = mapEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	return entries, nil
}

func (s *sqliteClient) getNotesForMonth(month, year int) (note, error) {
	q := `SELECT * FROM notes WHERE Month = ? AND Year = ?;`
	row := s.db.QueryRow(q, month, year)
	return mapNote(row)
}

func mapEntry(row *sql.Rows) (entry, error) {
	var e entry
	err := row.Scan(&e.Day, &e.Month, &e.Year, &e.State)
	return e, err
}

func mapNote(row *sql.Row) (note, error) {
	var n note
	err := row.Scan(&n.Month, &n.Year, &n.Notes)
	if err == sql.ErrNoRows {
		return note{}, nil
	}
	return n, err
}

func mapToModel(entries []entry, n note) models.Entry {
	if len(entries) == 0 {
		return models.Entry{}
	}

	userID := fmt.Sprintf("%d", entries[0].UserID)

	e := models.Entry{
		User:  userID,
		Month: entries[0].Month,
		Year:  entries[0].Year,
		Days:  make(map[string]int),
		Notes: n.Notes,
	}

	for _, day := range entries {
		e.Days[fmt.Sprintf("%d", day.Day)] = day.State
	}

	return e
}

func mapFromModel(e models.Entry) ([]entry, note, error) {
	var entries []entry
	var notes note

	userID, err := strconv.Atoi(e.User)
	if err != nil {
		return nil, note{}, err
	}

	for day, state := range e.Days {
		dayInt, err := strconv.Atoi(day)
		if err != nil {
			return nil, notes, err
		}
		entries = append(entries, entry{
			UserID: userID,
			Day:    dayInt,
			Month:  e.Month,
			Year:   e.Year,
			State:  state,
		})
	}

	notes = note{
		UserID: userID,
		Month:  e.Month,
		Year:   e.Year,
		Notes:  e.Notes,
	}

	return entries, notes, nil
}
