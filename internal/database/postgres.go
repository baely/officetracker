package database

import (
	"database/sql"
	"fmt"
	"strconv"

	_ "github.com/lib/pq"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/models"
)

const (
	PqConnFormat = "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable"
)

type postgres struct {
	cfg config.Postgres
	db  *sql.DB
}

func NewPostgres(cfg config.Postgres) (Databaser, error) {
	pqConnStr := fmt.Sprintf(PqConnFormat, cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)
	db, err := sql.Open("postgres", pqConnStr)
	if err != nil {
		return nil, err
	}

	p := &postgres{
		cfg: cfg,
		db:  db,
	}

	return p, nil
}

func (p *postgres) SaveEntry(e models.Entry) error {
	entries, notes, err := mapFromModel(e)
	if err != nil {
		return fmt.Errorf("failed to map entry: %v", err)
	}

	for _, e2 := range entries {
		if err = p.insertEntry(e2); err != nil {
			return fmt.Errorf("failed to insert entry: %v", err)
		}
	}

	if err = p.insertNote(notes); err != nil {
		return fmt.Errorf("failed to insert note: %v", err)
	}

	return nil
}

func (p *postgres) GetEntries(userID string, month, year int) (models.Entry, error) {
	userIDint, _ := strconv.Atoi(userID)
	entries, err := p.getEntriesForMonth(userIDint, month, year)
	if err != nil {
		return models.Entry{}, err
	}

	notes, err := p.getNotesForMonth(userIDint, month, year)
	if err != nil {
		return models.Entry{}, err
	}

	return mapToModel(entries, notes), nil
}

func (p *postgres) GetAllEntries(userID string) ([]models.Entry, error) {
	m := make(map[string][]entry)

	userIDint, _ := strconv.Atoi(userID)
	entries, err := p.getAllEntries(userIDint)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		k := fmt.Sprintf("%d-%d", e.Month, e.Year)
		m[k] = append(m[k], e)
	}

	var res []models.Entry
	for _, v := range m {
		notes, err := p.getNotesForMonth(userIDint, v[0].Month, v[0].Year)
		if err != nil {
			return nil, err
		}
		res = append(res, mapToModel(v, notes))
	}

	return res, nil
}

func (p *postgres) GetEntriesForBankYear(userID string, bankYear int) ([]models.Entry, error) {
	startMonth, startYear := 10, bankYear-1
	endMonth, endYear := 9, bankYear

	var res []models.Entry

	for month := startMonth; month <= 12; month++ {
		e, err := p.GetEntries(userID, month, startYear)
		if err != nil {
			return nil, err
		}
		res = append(res, e)
	}

	for month := 1; month <= endMonth; month++ {
		e, err := p.GetEntries(userID, month, endYear)
		if err != nil {
			return nil, err
		}
		res = append(res, e)
	}

	return res, nil
}

func (p *postgres) GetUserByGHID(ghID string) (int, error) {
	q := `SELECT user_id FROM users WHERE gh_id = $1;`
	row := p.db.QueryRow(q, ghID)
	var id int
	err := row.Scan(&id)
	if err == sql.ErrNoRows {
		return 0, ErrNoUser
	}
	return id, err
}

func (p *postgres) GetUserBySecret(secret string) (int, error) {
	q := `SELECT user_id FROM secrets WHERE secret = $1 AND active;`
	row := p.db.QueryRow(q, secret)
	var id int
	err := row.Scan(&id)
	if err == sql.ErrNoRows {
		return 0, ErrNoUser
	}
	return id, err
}

func (p *postgres) GetUser(userID string) (int, error) {
	userIDint, _ := strconv.Atoi(userID)
	q := `SELECT user_id FROM users WHERE user_id = $1;`
	row := p.db.QueryRow(q, userIDint)
	var id int
	err := row.Scan(&id)
	if err == sql.ErrNoRows {
		return 0, ErrNoUser
	}
	return id, err
}

func (p *postgres) SaveUser(ghID string) (int, error) {
	q := `INSERT INTO users (gh_id) VALUES ($1) RETURNING user_id;`
	row := p.db.QueryRow(q, ghID)
	var id int
	err := row.Scan(&id)
	return id, err
}

func (p *postgres) insertEntry(e entry) error {
	q := `INSERT INTO entries (user_id, day, month, year, state) VALUES ($1, $2, $3, $4, $5) ON CONFLICT(user_id, day, month, year) DO UPDATE SET state=EXCLUDED.state;`
	_, err := p.db.Exec(q, e.UserID, e.Day, e.Month, e.Year, e.State)
	return err
}

func (p *postgres) insertNote(n note) error {
	q := `INSERT INTO notes (user_id, month, year, notes) VALUES ($1, $2, $3, $4) ON CONFLICT(user_id, month, year) DO UPDATE SET notes=EXCLUDED.notes;`
	_, err := p.db.Exec(q, n.UserID, n.Month, n.Year, n.Notes)
	return err
}

func (p *postgres) getEntriesForMonth(userID, month, year int) ([]entry, error) {
	q := `SELECT user_id, day, month, year, state FROM entries WHERE user_id = $1 AND month = $2 AND year = $3;`
	rows, err := p.db.Query(q, userID, month, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []entry

	for rows.Next() {

		var e entry
		e, err = mapPqEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	return entries, nil
}

func (p *postgres) getAllEntries(userID int) ([]entry, error) {
	q := `SELECT user_id, day, month, year, state FROM entries WHERE user_id = $1;`
	rows, err := p.db.Query(q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []entry
	for rows.Next() {
		var e entry
		e, err = mapPqEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	return entries, nil
}

func (p *postgres) getNotesForMonth(userID, month, year int) (note, error) {
	q := `SELECT user_id, month, year, notes FROM notes WHERE user_id = $1 AND month = $2 AND year = $3;`
	row := p.db.QueryRow(q, userID, month, year)
	return mapPqNote(row)
}

func mapPqEntry(row *sql.Rows) (entry, error) {
	var e entry
	err := row.Scan(&e.UserID, &e.Day, &e.Month, &e.Year, &e.State)
	return e, err
}

func mapPqNote(row *sql.Row) (note, error) {
	var n note
	err := row.Scan(&n.UserID, &n.Month, &n.Year, &n.Notes)
	if err == sql.ErrNoRows {
		return note{}, nil
	}
	return n, err
}
