package database

import (
	"context"
	"database/sql"
	"encoding/json"
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
	q := `SELECT month, day, state FROM entries WHERE user_id = $1 AND ((year = $2 AND month > 9) OR (year = $3 AND month <= 9));`
	yearState := model.YearState{
		Months: make(map[int]model.MonthState),
	}
	err := p.readOnlyTransaction(func(tx *sql.Tx) error {
		rows, err := tx.Query(q, userID, year-1, year)
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
	q := `SELECT month, notes FROM notes WHERE user_id = $1 AND ((year = $2 AND month > 9) OR (year = $3 AND month <= 9));`
	notes := make(map[int]model.Note)
	err := p.readOnlyTransaction(func(tx *sql.Tx) error {
		rows, err := tx.Query(q, userID, year-1, year)
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
	q := `SELECT user_id FROM gh_users WHERE gh_id = $1;`
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
			return ErrNoUser
		}
		return err
	})
	return id, err
}

func (p *postgres) GetUserLinkedAccounts(userID int) ([]model.LinkedAccount, error) {
	q := `SELECT sub, profile FROM auth0_users WHERE user_id = $1 ORDER BY sub;`
	var accounts []model.LinkedAccount
	err := p.readOnlyTransaction(func(tx *sql.Tx) error {
		rows, err := tx.Query(q, userID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var sub, profileJSON string
			if err := rows.Scan(&sub, &profileJSON); err != nil {
				return err
			}

			// Parse provider from subject (format: "provider|identifier")
			parts := strings.Split(sub, "|")
			provider := "unknown"
			if len(parts) == 2 {
				provider = parts[0]
			}

			// Parse nickname from profile JSON
			var profile map[string]interface{}
			nickname := ""
			if err := json.Unmarshal([]byte(profileJSON), &profile); err == nil {
				if nick, ok := profile["nickname"].(string); ok {
					nickname = nick
				}
			}

			accounts = append(accounts, model.LinkedAccount{
				Provider: provider,
				Nickname: nickname,
			})
		}
		return rows.Err()
	})
	return accounts, err
}

func (p *postgres) GetUserByAuth0Sub(sub string) (int, error) {
	q := `SELECT user_id FROM auth0_users WHERE sub = $1;`
	var id int
	err := p.readOnlyTransaction(func(tx *sql.Tx) error {
		row := tx.QueryRow(q, sub)
		err := row.Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoUser
		}
		return err
	})
	return id, err
}

func (p *postgres) SaveUserByAuth0Sub(sub string, profile string) (int, error) {
	var id int
	err := p.readWriteTransaction(func(tx *sql.Tx) error {
		// Create the user entry
		q1 := `INSERT INTO users DEFAULT VALUES RETURNING user_id;`
		row := tx.QueryRow(q1)
		if err := row.Scan(&id); err != nil {
			return err
		}

		// Add to auth0_users table
		q2 := `INSERT INTO auth0_users (sub, profile, user_id) VALUES ($1, $2, $3);`
		_, err := tx.Exec(q2, sub, profile, id)
		return err
	})
	return id, err
}

func (p *postgres) UpdateAuth0Profile(sub string, profile string) error {
	q := `UPDATE auth0_users SET profile = $1 WHERE sub = $2;`
	return p.readWriteTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(q, profile, sub)
		return err
	})
}

func (p *postgres) LinkAuth0Account(userID int, sub string, profile string) error {
	return p.readWriteTransaction(func(tx *sql.Tx) error {
		// Check if this Auth0 subject already exists
		var existingUserID int
		checkQ := `SELECT user_id FROM auth0_users WHERE sub = $1;`
		err := tx.QueryRow(checkQ, sub).Scan(&existingUserID)

		if err == nil {
			// Auth0 subject exists - check if it belongs to another user
			if existingUserID != userID {
				return fmt.Errorf("auth0 account already associated with another user")
			}

			// Auth0 subject exists and belongs to this user - update profile only
			updateQ := `UPDATE auth0_users SET profile = $1 WHERE sub = $2;`
			_, err = tx.Exec(updateQ, profile, sub)
			return err
		}

		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		// Auth0 subject doesn't exist - insert new association
		insertQ := `INSERT INTO auth0_users (sub, profile, user_id) VALUES ($1, $2, $3);`
		_, err = tx.Exec(insertQ, sub, profile, userID)
		return err
	})
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

func (p *postgres) GetThemePreferences(userID int) (model.ThemePreferences, error) {
	q := `SELECT theme, weather_enabled, time_based_enabled, location FROM user_preferences WHERE user_id = $1;`
	var prefs model.ThemePreferences

	err := p.readOnlyTransaction(func(tx *sql.Tx) error {
		row := tx.QueryRow(q, userID)
		var location sql.NullString
		err := row.Scan(&prefs.Theme, &prefs.WeatherEnabled, &prefs.TimeBasedEnabled, &location)
		if errors.Is(err, sql.ErrNoRows) {
			// Return default values if no preferences exist
			prefs = model.ThemePreferences{
				Theme:            "default",
				WeatherEnabled:   false,
				TimeBasedEnabled: false,
			}
			return nil
		}
		if location.Valid {
			prefs.Location = location.String
		}
		return err
	})

	return prefs, err
}

func (p *postgres) SaveThemePreferences(userID int, prefs model.ThemePreferences) error {
	q := `INSERT INTO user_preferences (user_id, theme, weather_enabled, time_based_enabled, location)
		  VALUES ($1, $2, $3, $4, $5)
		  ON CONFLICT (user_id)
		  DO UPDATE SET theme = $2, weather_enabled = $3, time_based_enabled = $4, location = $5;`

	return p.readWriteTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(q, userID, prefs.Theme, prefs.WeatherEnabled, prefs.TimeBasedEnabled, prefs.Location)
		return err
	})
}

func (p *postgres) GetSchedulePreferences(userID int) (model.SchedulePreferences, error) {
	q := `SELECT schedule_monday_state, schedule_tuesday_state, schedule_wednesday_state, schedule_thursday_state, 
		         schedule_friday_state, schedule_saturday_state, schedule_sunday_state 
		  FROM user_preferences WHERE user_id = $1;`

	var prefs model.SchedulePreferences
	err := p.readOnlyTransaction(func(tx *sql.Tx) error {
		row := tx.QueryRow(q, userID)
		err := row.Scan(&prefs.Monday, &prefs.Tuesday, &prefs.Wednesday, &prefs.Thursday,
			&prefs.Friday, &prefs.Saturday, &prefs.Sunday)
		if errors.Is(err, sql.ErrNoRows) {
			// Return default values if no preferences exist
			prefs = model.SchedulePreferences{
				Monday:    model.StateUntracked,
				Tuesday:   model.StateUntracked,
				Wednesday: model.StateUntracked,
				Thursday:  model.StateUntracked,
				Friday:    model.StateUntracked,
				Saturday:  model.StateUntracked,
				Sunday:    model.StateUntracked,
			}
			return nil
		}
		return err
	})

	return prefs, err
}

func (p *postgres) SaveSchedulePreferences(userID int, prefs model.SchedulePreferences) error {
	q := `INSERT INTO user_preferences (user_id, schedule_monday_state, schedule_tuesday_state, schedule_wednesday_state, 
		         schedule_thursday_state, schedule_friday_state, schedule_saturday_state, schedule_sunday_state)
		  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		  ON CONFLICT (user_id)
		  DO UPDATE SET schedule_monday_state = $2, schedule_tuesday_state = $3, schedule_wednesday_state = $4,
		                schedule_thursday_state = $5, schedule_friday_state = $6, schedule_saturday_state = $7, schedule_sunday_state = $8;`

	return p.readWriteTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(q, userID, int(prefs.Monday), int(prefs.Tuesday), int(prefs.Wednesday),
			int(prefs.Thursday), int(prefs.Friday), int(prefs.Saturday), int(prefs.Sunday))
		return err
	})
}

func (p *postgres) IsUserSuspended(userID int) (bool, error) {
	q := `SELECT suspended FROM users WHERE user_id = $1;`
	var suspended bool
	err := p.readOnlyTransaction(func(tx *sql.Tx) error {
		row := tx.QueryRow(q, userID)
		err := row.Scan(&suspended)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	})
	return suspended, err
}
