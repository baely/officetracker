package database

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/pkg/model"
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

func (p *postgres) SaveDay(userID int, day int, month int, year int, state model.DayState) error {
	q := `INSERT INTO entries (user_id, day, month, year, state) VALUES ($1, $2, $3, $4, $5) ON CONFLICT(user_id, day, month, year) DO UPDATE SET state=EXCLUDED.state;`
	_, err := p.db.Exec(q, userID, day, month, year, state.State)
	return err
}

func (p *postgres) GetDay(userID int, day int, month int, year int) (model.DayState, error) {
	q := `SELECT state FROM entries WHERE user_id = $1 AND day = $2 AND month = $3 AND year = $4;`
	row := p.db.QueryRow(q, userID, day, month, year)
	var state model.DayState
	err := row.Scan(&(state.State))
	if err == sql.ErrNoRows {
		return state, nil
	}
	return state, err
}

func (p *postgres) SaveMonth(userID int, month int, year int, state model.MonthState) error {
	argNum := incrementer(1)
	q := `INSERT INTO entries (user_id, day, month, year, state) VALUES `
	var queries []string
	var args []interface{}
	for day, dayState := range state.Days {
		q += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d), ", argNum(), argNum(), argNum(), argNum(), argNum())
		args = append(args, userID, day, month, year, dayState.State)
	}
	q = q + strings.Join(queries, ", ") + " ON CONFLICT(user_id, day, month, year) DO UPDATE SET state=EXCLUDED.state;"
	_, err := p.db.Exec(q, args...)
	return err
}

func (p *postgres) GetMonth(userID int, month int, year int) (model.MonthState, error) {
	q := `SELECT day, state FROM entries WHERE user_id = $1 AND month = $2 AND year = $3;`
	rows, err := p.db.Query(q, userID, month, year)
	if err != nil {
		return model.MonthState{}, err
	}
	defer rows.Close()
	monthState := model.MonthState{
		Days: make(map[int]model.DayState),
	}
	for rows.Next() {
		var day int
		var dayState model.DayState
		err = rows.Scan(&day, &dayState.State)
		if err != nil {
			return model.MonthState{}, err
		}
		monthState.Days[day] = dayState
	}
	return monthState, nil
}

func (p *postgres) GetYear(userID int, year int) (model.YearState, error) {
	q := `SELECT month, day, state FROM entries WHERE user_id = $1 AND year = $2;`
	rows, err := p.db.Query(q, userID, year)
	if err != nil {
		return model.YearState{}, err
	}
	defer rows.Close()
	yearState := model.YearState{
		Months: make(map[int]model.MonthState),
	}
	for rows.Next() {
		var month, day int
		var dayState model.DayState
		err = rows.Scan(&month, &day, &dayState.State)
		if err != nil {
			return model.YearState{}, err
		}
		if _, ok := yearState.Months[month]; !ok {
			yearState.Months[month] = model.MonthState{
				Days: make(map[int]model.DayState),
			}
		}
		yearState.Months[month].Days[day] = dayState
	}
	return yearState, nil
}

func (p *postgres) SaveNote(userID int, month int, year int, note string) error {
	q := `INSERT INTO notes (user_id, month, year, notes) VALUES ($1, $2, $3, $4) ON CONFLICT(user_id, month, year) DO UPDATE SET notes=EXCLUDED.notes;`
	_, err := p.db.Exec(q, userID, month, year, note)
	return err
}

func (p *postgres) GetNote(userID int, month int, year int) (string, error) {
	q := `SELECT notes FROM notes WHERE user_id = $1 AND month = $2 AND year = $3;`
	row := p.db.QueryRow(q, userID, month, year)
	var noteString string
	err := row.Scan(&noteString)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return noteString, err
}

func (p *postgres) GetUser(userID int) (int, error) {
	q := `SELECT user_id FROM users WHERE user_id = $1;`
	row := p.db.QueryRow(q, userID)
	var id int
	err := row.Scan(&id)
	if err == sql.ErrNoRows {
		return 0, ErrNoUser
	}
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (p *postgres) SaveUserByGHID(ghID string) (int, error) {
	q := `INSERT INTO users (gh_id) VALUES ($1) RETURNING user_id;`
	row := p.db.QueryRow(q, ghID)
	var id int
	err := row.Scan(&id)
	return id, err
}

func (p *postgres) SaveSecret(userID int, secret string) error {
	q := `UPDATE secrets SET active = false WHERE user_id = $1 AND active;`
	_, err := p.db.Exec(q, userID)
	if err != nil {
		return err
	}
	q = `INSERT INTO secrets (user_id, secret, active) VALUES ($1, $2, true);`
	_, err = p.db.Exec(q, userID, secret)
	return err
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

func incrementer(start int) func() int {
	i := start
	return func() int {
		x := i
		i++
		return x
	}
}
