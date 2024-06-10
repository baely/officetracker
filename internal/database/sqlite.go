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

func (s *sqliteClient) SaveDay(userID int, day int, month int, year int, state model.DayState) error {
	//TODO implement me
	panic("implement me")
}

func (s *sqliteClient) GetDay(userID int, day int, month int, year int) (model.DayState, error) {
	//TODO implement me
	panic("implement me")
}

func (s *sqliteClient) SaveMonth(userID int, month int, year int, state model.MonthState) error {
	//TODO implement me
	panic("implement me")
}

func (s *sqliteClient) GetMonth(userID int, month int, year int) (model.MonthState, error) {
	//TODO implement me
	panic("implement me")
}

func (s *sqliteClient) GetYear(userID int, year int) (model.YearState, error) {
	//TODO implement me
	panic("implement me")
}

func (s *sqliteClient) SaveNote(userID int, month int, year int, note string) error {
	//TODO implement me
	panic("implement me")
}

func (s *sqliteClient) GetNote(userID int, month int, year int) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (s *sqliteClient) GetUser(userID int) (int, error) {
	//TODO implement me
	panic("implement me")
}

func (s *sqliteClient) SaveUserByGHID(ghID string) (int, error) {
	//TODO implement me
	panic("implement me")
}

func (s *sqliteClient) SaveSecret(userID int, secret string) error {
	//TODO implement me
	panic("implement me")
}

func (s *sqliteClient) GetUserByGHID(_ string) (int, error) {
	return 1, nil
}

func (s *sqliteClient) GetUserBySecret(_ string) (int, error) {
	return 1, nil
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
