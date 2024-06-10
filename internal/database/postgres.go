package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

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
	return p.readWriteTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(q, userID, day, month, year, state.State)
		return err
	})
}

func (p *postgres) GetDay(userID int, day int, month int, year int) (model.DayState, error) {
	q := `SELECT state FROM entries WHERE user_id = $1 AND day = $2 AND month = $3 AND year = $4;`
	var state model.DayState
	err := p.readOnlyTransaction(func(tx *sql.Tx) error {
		row := tx.QueryRow(q, userID, day, month, year)
		err := row.Scan(&(state.State))
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	})
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
	err := p.readWriteTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(q, args...)
		return err
	})
	return err
}

func (p *postgres) GetMonth(userID int, month int, year int) (model.MonthState, error) {
	q := `SELECT day, state FROM entries WHERE user_id = $1 AND month = $2 AND year = $3;`
	var monthState model.MonthState
	err := p.readOnlyTransaction(func(tx *sql.Tx) error {
		rows, err := tx.Query(q, userID, month, year)
		if err != nil {
			return err
		}
		defer rows.Close()
		monthState.Days = make(map[int]model.DayState)
		for rows.Next() {
			var day int
			var dayState model.DayState
			err = rows.Scan(&day, &dayState.State)
			if err != nil {
				return err
			}
			monthState.Days[day] = dayState
		}
		return rows.Err()
	})
	return monthState, err
}

func (p *postgres) GetYear(userID int, year int) (model.YearState, error) {
	q := `SELECT month, day, state FROM entries WHERE user_id = $1 AND year = $2;`
	yearState := model.YearState{
		Months: make(map[int]model.MonthState),
	}
	err := p.readOnlyTransaction(func(tx *sql.Tx) error {
		rows, err := tx.Query(q, userID, year)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var month, day int
			var dayState model.DayState
			err = rows.Scan(&month, &day, &dayState.State)
			if err != nil {
				return err
			}
			if _, ok := yearState.Months[month]; !ok {
				yearState.Months[month] = model.MonthState{
					Days: make(map[int]model.DayState),
				}
			}
			yearState.Months[month].Days[day] = dayState
		}
		return rows.Err()
	})
	return yearState, err
}

func (p *postgres) SaveNote(userID int, month int, year int, note string) error {
	q := `INSERT INTO notes (user_id, month, year, notes) VALUES ($1, $2, $3, $4) ON CONFLICT(user_id, month, year) DO UPDATE SET notes=EXCLUDED.notes;`
	err := p.readWriteTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(q, userID, month, year, note)
		return err
	})
	return err
}

func (p *postgres) GetNote(userID int, month int, year int) (model.Note, error) {
	q := `SELECT notes FROM notes WHERE user_id = $1 AND month = $2 AND year = $3;`
	var noteModel model.Note
	err := p.readOnlyTransaction(func(tx *sql.Tx) error {
		row := tx.QueryRow(q, userID, month, year)
		err := row.Scan(&noteModel.Note)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	})
	return noteModel, err
}

func (p *postgres) GetNotes(userID int, year int) (map[int]model.Note, error) {
	q := `SELECT month, notes FROM notes WHERE user_id = $1 AND year = $2;`
	notes := make(map[int]model.Note)
	err := p.readOnlyTransaction(func(tx *sql.Tx) error {
		rows, err := tx.Query(q, userID, year)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var month int
			var noteModel model.Note
			err = rows.Scan(&month, &noteModel.Note)
			if err != nil {
				return err
			}
			notes[month] = noteModel
		}
		return rows.Err()
	})
	return notes, err

}

func (p *postgres) GetUser(userID int) (int, error) {
	q := `SELECT user_id FROM users WHERE user_id = $1;`
	var id int
	err := p.readOnlyTransaction(func(tx *sql.Tx) error {
		row := tx.QueryRow(q, userID)
		err := row.Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	})
	return id, err
}

func (p *postgres) SaveUserByGHID(ghID string) (int, error) {
	q := `INSERT INTO users (gh_id) VALUES ($1) RETURNING user_id;`
	var id int
	err := p.readWriteTransaction(func(tx *sql.Tx) error {
		row := tx.QueryRow(q, ghID)
		err := row.Scan(&id)
		return err
	})
	return id, err
}

func (p *postgres) SaveSecret(userID int, secret string) error {
	q := `UPDATE secrets SET active = false WHERE user_id = $1 AND active;`
	err := p.readWriteTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(q, userID)
		return err
	})
	if err != nil {
		return err
	}
	q = `INSERT INTO secrets (user_id, secret, active) VALUES ($1, $2, true);`
	err = p.readWriteTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(q, userID, secret)
		return err
	})
	return err
}

func (p *postgres) GetUserByGHID(ghID string) (int, error) {
	q := `SELECT user_id FROM users WHERE gh_id = $1;`
	var id int
	err := p.readOnlyTransaction(func(tx *sql.Tx) error {
		row := tx.QueryRow(q, ghID)
		err := row.Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoUser
		}
		return err
	})
	return id, err
}

func (p *postgres) GetUserBySecret(secret string) (int, error) {
	q := `SELECT user_id FROM secrets WHERE secret = $1 AND active;`
	var id int
	err := p.readOnlyTransaction(func(tx *sql.Tx) error {
		row := tx.QueryRow(q, secret)
		err := row.Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	})
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

func (p *postgres) rcvTx(fn func(*sql.Tx) error, opts *sql.TxOptions) error {
	ctx := context.Background()
	start := time.Now()
	defer func() {
		slog.Info(fmt.Sprintf("transaction took: %s", time.Since(start)))
	}()
	conn, err := p.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, opts)
	if err != nil {
		return err
	}
	err = fn(tx)
	commitErr := tx.Commit()
	if commitErr != nil {
		err = commitErr
	}
	return err
}

func (p *postgres) readOnlyTransaction(fn func(*sql.Tx) error) error {
	opts := &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  true,
	}
	return p.rcvTx(fn, opts)
}

func (p *postgres) readWriteTransaction(fn func(*sql.Tx) error) error {
	opts := &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	}
	return p.rcvTx(fn, opts)
}
