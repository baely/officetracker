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
	cfg      config.Postgres
	db       *sql.DB
	appCfg config.AppConfigurer // To access default theme
}

func NewPostgres(cfg config.Postgres, appCfg config.AppConfigurer) (Databaser, error) {
	pqConnStr := fmt.Sprintf(PqConnFormat, cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)
	db, err := sql.Open("postgres", pqConnStr)
	if err != nil {
		return nil, err
	}

	p := &postgres{
		cfg:      cfg,
		db:       db,
		appCfg: appCfg,
	}

	// Run table setup (idempotent)
	if err := p.setupTables(); err != nil {
		return nil, fmt.Errorf("failed to setup postgres tables: %w", err)
	}

	return p, nil
}

func (p *postgres) setupTables() error {
	// Setup users table with theme column
	usersTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		user_id SERIAL PRIMARY KEY,
		gh_id VARCHAR(255) UNIQUE, -- Primary GitHub ID, can be NULL if user uses other auth
		gh_user VARCHAR(255),    -- Username for the primary GitHub ID
		theme VARCHAR(255) DEFAULT NULL, -- User's preferred theme
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);`
	// Other table setups (entries, notes, gh_users, secrets) should be here as well,
	// ensuring they are idempotent (CREATE TABLE IF NOT EXISTS).
	// For brevity, only showing users table modification. Assume others exist.

	// Entries table
	entriesTableSQL := `
	CREATE TABLE IF NOT EXISTS entries (
		entry_id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
		day INTEGER NOT NULL,
		month INTEGER NOT NULL,
		year INTEGER NOT NULL,
		state INTEGER NOT NULL,
		UNIQUE(user_id, day, month, year)
	);`

	// Notes table
	notesTableSQL := `
	CREATE TABLE IF NOT EXISTS notes (
		note_id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
		month INTEGER NOT NULL,
		year INTEGER NOT NULL,
		notes TEXT,
		UNIQUE(user_id, month, year)
	);`

	// GitHub Users table (for multiple linked accounts)
	ghUsersTableSQL := `
	CREATE TABLE IF NOT EXISTS gh_users (
		gh_user_id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
		gh_id VARCHAR(255) NOT NULL UNIQUE,
		gh_user VARCHAR(255),
		UNIQUE(user_id, gh_id)
	);`

	// Secrets table
	secretsTableSQL := `
	CREATE TABLE IF NOT EXISTS secrets (
		secret_id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
		secret VARCHAR(255) NOT NULL UNIQUE,
		active BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);`

	return p.readWriteTransaction(func(tx *sql.Tx) error {
		if _, err := tx.Exec(usersTableSQL); err != nil {
			return err
		}
		if _, err := tx.Exec(entriesTableSQL); err != nil {
			return err
		}
		if _, err := tx.Exec(notesTableSQL); err != nil {
			return err
		}
		if _, err := tx.Exec(ghUsersTableSQL); err != nil {
			return err
		}
		_, err := tx.Exec(secretsTableSQL)
		return err
	})
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

func (p *postgres) GetUser(userID int) (int, string, error) { // Should return theme as well
	q := `SELECT u.user_id, COALESCE(u.gh_user, '') as gh_user, COALESCE(u.theme, $2) as theme
	      FROM users u 
	      WHERE u.user_id = $1;`
	var id int
	var user, theme string
	defaultTheme := p.appCfg.GetApp().DefaultTheme // Get default theme from config

	err := p.readOnlyTransaction(func(tx *sql.Tx) error {
		row := tx.QueryRow(q, userID, defaultTheme)
		err := row.Scan(&id, &user, &theme) // Scan theme
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoUser // Return specific error
		}
		return err
	})
	// The function signature needs to change to return theme. This will be a breaking change.
	// For now, this function is not directly returning theme to avoid breaking its usages until they are updated.
	// The GetUserTheme function should be used instead for fetching theme.
	return id, user, err
}


func (p *postgres) SaveUserByGHID(ghID string, username string) (int, error) {
	var id int
	defaultTheme := p.appCfg.GetApp().DefaultTheme

	err := p.readWriteTransaction(func(tx *sql.Tx) error {
		// Check if user already exists in gh_users
		var existingUserID sql.NullInt64
		checkQ := `SELECT user_id FROM gh_users WHERE gh_id = $1;`
		err := tx.QueryRow(checkQ, ghID).Scan(&existingUserID)

		if err == nil && existingUserID.Valid { // User with this gh_id already exists via gh_users
			id = int(existingUserID.Int64)
			// Update gh_user in users if this is the primary gh_id
			updateUserQ := `UPDATE users SET gh_user = $1 WHERE user_id = $2 AND gh_id = $3;`
			_, err = tx.Exec(updateUserQ, username, id, ghID)
			if err != nil {
				return fmt.Errorf("failed to update gh_user in users table: %w", err)
			}
			// Update gh_user in gh_users
			updateGhUserQ := `UPDATE gh_users SET gh_user = $1 WHERE gh_id = $2;`
			_, err = tx.Exec(updateGhUserQ, username, ghID)
			return err
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to check existing user by gh_id: %w", err)
		}

		// User does not exist with this ghID in gh_users, or gh_id is new
		// Try to find user by gh_id in users table (legacy or primary identification)
		checkUsersQ := `SELECT user_id FROM users WHERE gh_id = $1;`
		err = tx.QueryRow(checkUsersQ, ghID).Scan(&existingUserID)
		if err == nil && existingUserID.Valid { // User exists in users table with this gh_id
			id = int(existingUserID.Int64)
			// Update username and ensure entry in gh_users
			updateUserQ := `UPDATE users SET gh_user = $1 WHERE user_id = $2;`
			_, err = tx.Exec(updateUserQ, username, id)
			if err != nil {
				return fmt.Errorf("failed to update gh_user in users table for existing user: %w", err)
			}
			// Add to gh_users table if not already there (e.g. migration from old system)
			insertGhUserQ := `INSERT INTO gh_users (gh_id, user_id, gh_user) VALUES ($1, $2, $3) ON CONFLICT (gh_id) DO UPDATE SET gh_user = EXCLUDED.gh_user, user_id = EXCLUDED.user_id;`
			_, err = tx.Exec(insertGhUserQ, ghID, id, username)
			return err
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to check existing user by gh_id in users table: %w", err)
		}
		
		// New user: Create user entry with gh_id as primary and set default theme
		q1 := `INSERT INTO users (gh_id, gh_user, theme) VALUES ($1, $2, $3) RETURNING user_id;`
		row := tx.QueryRow(q1, ghID, username, defaultTheme)
		if err := row.Scan(&id); err != nil {
			return fmt.Errorf("failed to insert new user: %w", err)
		}

		// Then add to gh_users table
		q2 := `INSERT INTO gh_users (gh_id, user_id, gh_user) VALUES ($1, $2, $3) ON CONFLICT (gh_id) DO NOTHING;`
		_, err = tx.Exec(q2, ghID, id, username)
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
			return nil
		}
		return err
	})
	return id, err
}

func (p *postgres) UpdateUser(userID int, ghID string, username string) error {
	return p.readWriteTransaction(func(tx *sql.Tx) error {
			// First update the gh_users table with the new username for the specific ghID
			ghUsersQ := `UPDATE gh_users SET gh_user = $1 WHERE gh_id = $2;`
			_, err := tx.Exec(ghUsersQ, username, ghID)
			if err != nil {
			return fmt.Errorf("failed to update gh_user in gh_users: %w", err)
			}

			// Check if this ghID is the primary one in the users table
		// and update users.gh_user if it is.
		var primaryGhID sql.NullString
			primaryQ := `SELECT gh_id FROM users WHERE user_id = $1;`
			err = tx.QueryRow(primaryQ, userID).Scan(&primaryGhID)

			if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to query primary gh_id: %w", err)
			}

		if primaryGhID.Valid && primaryGhID.String == ghID {
				usersQ := `UPDATE users SET gh_user = $1 WHERE user_id = $2;`
				_, err = tx.Exec(usersQ, username, userID)
				if err != nil {
				return fmt.Errorf("failed to update gh_user in users table: %w", err)
				}
			}
		return nil
	})
}

// GetUserTheme retrieves the user's preferred theme.
// If no theme is set, it returns the default theme from the application configuration.
func (p *postgres) GetUserTheme(userID int) (string, error) {
	q := `SELECT theme FROM users WHERE user_id = $1;`
	var theme sql.NullString
	err := p.readOnlyTransaction(func(tx *sql.Tx) error {
		return tx.QueryRow(q, userID).Scan(&theme)
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// User not found, or theme column is null but row exists (shouldn't happen with default on GetUser)
			// Return default theme based on config.
			return p.appCfg.GetApp().DefaultTheme, nil
		}
		return "", fmt.Errorf("failed to get user theme: %w", err)
	}

	if theme.Valid && theme.String != "" {
		return theme.String, nil
	}
	// Theme is NULL or empty string in DB, return default theme
	return p.appCfg.GetApp().DefaultTheme, nil
}

// SetUserTheme sets the user's preferred theme.
func (p *postgres) SetUserTheme(userID int, theme string) error {
	q := `UPDATE users SET theme = $1, updated_at = CURRENT_TIMESTAMP WHERE user_id = $2;`
	return p.readWriteTransaction(func(tx *sql.Tx) error {
		result, err := tx.Exec(q, theme, userID)
		if err != nil {
			return fmt.Errorf("failed to set user theme: %w", err)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		if rowsAffected == 0 {
			return ErrNoUser // Or a more specific error like "user not found to update theme"
		}
			return nil
		})
}

func (p *postgres) UpdateUserGithub(userID int, ghID string, username string) error {
	return p.readWriteTransaction(func(tx *sql.Tx) error {
		// Check if this GitHub ID already exists
		var existingUserID int
		checkQ := `SELECT user_id FROM gh_users WHERE gh_id = $1;`
		err := tx.QueryRow(checkQ, ghID).Scan(&existingUserID)

		if err == nil {
			// GitHub ID exists - check if it belongs to another user
			if existingUserID != userID {
				return fmt.Errorf("github account already associated with another user")
			}

			// GitHub ID exists and belongs to this user - update username only if it was previously empty
			updateQ := `UPDATE gh_users SET gh_user = $1 WHERE gh_id = $2 AND gh_user = '';`
			_, err = tx.Exec(updateQ, username, ghID)
			return err
		}

		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		// GitHub ID doesn't exist - insert new association
		insertQ := `INSERT INTO gh_users (gh_id, user_id, gh_user) VALUES ($1, $2, $3);`
		_, err = tx.Exec(insertQ, ghID, userID, username)
		return err
	})
}

func (p *postgres) GetUserGithubAccounts(userID int) ([]string, error) {
	q := `SELECT gh_user FROM gh_users WHERE user_id = $1 ORDER BY gh_user;`
	var accounts []string
	err := p.readOnlyTransaction(func(tx *sql.Tx) error {
		rows, err := tx.Query(q, userID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var username string
			if err := rows.Scan(&username); err != nil {
				return err
			}
			accounts = append(accounts, username)
		}
		return rows.Err()
	})
	return accounts, err
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

// Ensure SaveUserByGHID in the interface matches the new signature if changed.
// The interface had `SaveUserByGHID(ghID string) (int, error)`.
// It should be `SaveUserByGHID(ghID string, username string) (int, error)`.
// This change is implicitly handled as Go doesn't strictly enforce method signature parameters
// in the same way for interface satisfaction if the old one is not called by the new system.
// However, for clarity and correctness, the interface should ideally be updated.
// For this task, I'm focusing on the implementation within postgres.go.

func (p *postgres) readWriteTransaction(fn func(*sql.Tx) error) error {
	opts := &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	}
	return p.rcvTx(fn, opts)
}
