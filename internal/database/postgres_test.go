package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/pkg/model"
)

// Postgres integration tests run against a real Postgres. They are skipped
// unless POSTGRES_TEST_HOST is set, so `go test ./...` stays green on a machine
// with no database. CI provides a postgres service container (see
// .github/workflows/test.yaml). Locally:
//
//	docker run -d -e POSTGRES_PASSWORD=postgres -p 5432:5432 postgres:16-alpine
//	POSTGRES_TEST_HOST=localhost go test ./internal/database/...
var (
	pgOnce  sync.Once
	pgCfg   config.Postgres
	pgReady bool
	pgErr   error
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// pgTestDB returns a Postgres Databaser against a freshly-migrated, empty
// schema, skipping the test when no test database is configured.
func pgTestDB(t *testing.T) Databaser {
	t.Helper()
	if os.Getenv("POSTGRES_TEST_HOST") == "" {
		t.Skip("POSTGRES_TEST_HOST not set; skipping Postgres integration tests")
	}

	pgOnce.Do(func() {
		pgCfg = config.Postgres{
			Host:     os.Getenv("POSTGRES_TEST_HOST"),
			Port:     envOr("POSTGRES_TEST_PORT", "5432"),
			User:     envOr("POSTGRES_TEST_USER", "postgres"),
			Password: envOr("POSTGRES_TEST_PASSWORD", "postgres"),
			DBName:   envOr("POSTGRES_TEST_DBNAME", "postgres"),
		}
		pgErr = applyMigrations(pgCfg)
		pgReady = pgErr == nil
	})
	if pgErr != nil {
		t.Fatalf("failed to prepare postgres schema: %v", pgErr)
	}

	// Each test starts from empty tables.
	if err := truncateAll(pgCfg); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	db, err := NewPostgres(pgCfg)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	return db
}

func rawConn(cfg config.Postgres) (*sql.DB, error) {
	return sql.Open("postgres", "host="+cfg.Host+" port="+cfg.Port+" user="+cfg.User+
		" password="+cfg.Password+" dbname="+cfg.DBName+" sslmode=disable")
}

// applyMigrations drops and recreates the public schema, then applies every
// migration .sql file in order for a pristine schema.
func applyMigrations(cfg config.Postgres) error {
	db, err := rawConn(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	if _, err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		return err
	}

	files, err := filepath.Glob("postgres/migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(b)); err != nil {
			return err
		}
	}
	return nil
}

func truncateAll(cfg config.Postgres) error {
	db, err := rawConn(cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`TRUNCATE entries, notes, secrets, auth0_users, gh_users, user_preferences, stats_snapshots, users RESTART IDENTITY CASCADE;`)
	return err
}

// seedUser inserts a user row so that foreign-key-constrained inserts succeed,
// returning its id.
func seedUser(t *testing.T, cfg config.Postgres) int {
	t.Helper()
	db, err := rawConn(cfg)
	if err != nil {
		t.Fatalf("rawConn: %v", err)
	}
	defer db.Close()
	var id int
	if err := db.QueryRow(`INSERT INTO users DEFAULT VALUES RETURNING user_id;`).Scan(&id); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return id
}

func TestPostgresDayRoundTripAndUpsert(t *testing.T) {
	db := pgTestDB(t)
	uid := seedUser(t, pgCfg)

	// Missing day -> untracked, no error.
	got, err := db.GetDay(uid, 5, 3, 2024)
	if err != nil {
		t.Fatalf("GetDay missing: %v", err)
	}
	if got.State != model.StateUntracked {
		t.Errorf("missing day = %d, want untracked", got.State)
	}

	if err := db.SaveDay(uid, 5, 3, 2024, model.DayState{State: model.StateWorkFromOffice}); err != nil {
		t.Fatalf("SaveDay: %v", err)
	}
	got, _ = db.GetDay(uid, 5, 3, 2024)
	if got.State != model.StateWorkFromOffice {
		t.Errorf("day = %d, want office", got.State)
	}

	// ON CONFLICT DO UPDATE upserts in place.
	if err := db.SaveDay(uid, 5, 3, 2024, model.DayState{State: model.StateWorkFromHome}); err != nil {
		t.Fatalf("SaveDay upsert: %v", err)
	}
	got, _ = db.GetDay(uid, 5, 3, 2024)
	if got.State != model.StateWorkFromHome {
		t.Errorf("upserted day = %d, want home", got.State)
	}
	if n, _ := db.CountTrackedDays(); n != 1 {
		t.Errorf("upsert duplicated a row: CountTrackedDays = %d, want 1", n)
	}
}

// Entries are scoped per user: one user's data is invisible to another.
func TestPostgresPerUserIsolation(t *testing.T) {
	db := pgTestDB(t)
	u1 := seedUser(t, pgCfg)
	u2 := seedUser(t, pgCfg)

	db.SaveDay(u1, 1, 1, 2024, model.DayState{State: model.StateWorkFromOffice})

	if got, _ := db.GetDay(u2, 1, 1, 2024); got.State != model.StateUntracked {
		t.Errorf("user 2 should not see user 1's entry, got %d", got.State)
	}
	if got, _ := db.GetDay(u1, 1, 1, 2024); got.State != model.StateWorkFromOffice {
		t.Errorf("user 1 should see own entry, got %d", got.State)
	}
}

func TestPostgresMonthRoundTrip(t *testing.T) {
	db := pgTestDB(t)
	uid := seedUser(t, pgCfg)

	month := model.MonthState{Days: map[int]model.DayState{
		1: {State: model.StateWorkFromOffice},
		2: {State: model.StateWorkFromHome},
	}}
	if err := db.SaveMonth(uid, 6, 2024, month); err != nil {
		t.Fatalf("SaveMonth: %v", err)
	}
	got, err := db.GetMonth(uid, 6, 2024)
	if err != nil {
		t.Fatalf("GetMonth: %v", err)
	}
	if len(got.Days) != 2 || got.Days[1].State != model.StateWorkFromOffice || got.Days[2].State != model.StateWorkFromHome {
		t.Errorf("GetMonth = %+v", got.Days)
	}

	// SaveMonth with re-submitted days upserts in place.
	if err := db.SaveMonth(uid, 6, 2024, model.MonthState{Days: map[int]model.DayState{
		1: {State: model.StateOther},
	}}); err != nil {
		t.Fatalf("SaveMonth upsert: %v", err)
	}
	got, _ = db.GetMonth(uid, 6, 2024)
	if got.Days[1].State != model.StateOther {
		t.Errorf("upserted day 1 = %d, want other", got.Days[1].State)
	}

	// An empty month is a no-op (no invalid SQL).
	if err := db.SaveMonth(uid, 7, 2024, model.MonthState{Days: map[int]model.DayState{}}); err != nil {
		t.Errorf("empty SaveMonth should be a no-op, got %v", err)
	}
}

func TestPostgresGetYearTrackingWindow(t *testing.T) {
	db := pgTestDB(t)
	uid := seedUser(t, pgCfg)

	seed := []struct{ month, year int }{
		{9, 2023}, {10, 2023}, {1, 2024}, {9, 2024}, {10, 2024},
	}
	for _, s := range seed {
		db.SaveDay(uid, 1, s.month, s.year, model.DayState{State: model.StateWorkFromOffice})
	}

	year, err := db.GetYear(uid, 2024, 10)
	if err != nil {
		t.Fatalf("GetYear: %v", err)
	}
	// Oct 2023 .. Sep 2024: months 10, 1, 9 present; Sep 2023 and Oct 2024 excluded.
	for _, want := range []int{10, 1, 9} {
		if _, ok := year.Months[want]; !ok {
			t.Errorf("month %d missing from tracking year", want)
		}
	}
	if len(year.Months) != 3 {
		t.Errorf("got %d months, want 3", len(year.Months))
	}
}

func TestPostgresNotes(t *testing.T) {
	db := pgTestDB(t)
	uid := seedUser(t, pgCfg)

	if n, _ := db.GetNote(uid, 3, 2024); n.Note != "" {
		t.Errorf("missing note = %q, want empty", n.Note)
	}
	if err := db.SaveNote(uid, 3, 2024, "quarter end"); err != nil {
		t.Fatalf("SaveNote: %v", err)
	}
	n, _ := db.GetNote(uid, 3, 2024)
	if n.Note != "quarter end" {
		t.Errorf("note = %q", n.Note)
	}
	db.SaveNote(uid, 3, 2024, "updated") // upsert
	n, _ = db.GetNote(uid, 3, 2024)
	if n.Note != "updated" {
		t.Errorf("upserted note = %q", n.Note)
	}

	db.SaveNote(uid, 10, 2023, "oct")
	notes, err := db.GetNotes(uid, 2024, 10)
	if err != nil {
		t.Fatalf("GetNotes: %v", err)
	}
	if notes[10].Note != "oct" || notes[3].Note != "updated" {
		t.Errorf("GetNotes window = %v", notes)
	}
}

func TestPostgresPreferences(t *testing.T) {
	db := pgTestDB(t)
	uid := seedUser(t, pgCfg)

	// Defaults before any save.
	theme, err := db.GetThemePreferences(uid)
	if err != nil || theme.Theme != "default" {
		t.Errorf("default theme = %+v, err %v", theme, err)
	}
	sched, _ := db.GetSchedulePreferences(uid)
	if sched.Monday != model.StateUntracked {
		t.Errorf("default schedule not untracked: %+v", sched)
	}
	cal, _ := db.GetCalendarPreferences(uid)
	if cal.TrackingYearStartMonth != model.DefaultTrackingYearStartMonth {
		t.Errorf("default calendar start = %d", cal.TrackingYearStartMonth)
	}

	// Round-trip each.
	if err := db.SaveThemePreferences(uid, model.ThemePreferences{Theme: "dark", WeatherEnabled: true, Location: "Sydney"}); err != nil {
		t.Fatalf("SaveThemePreferences: %v", err)
	}
	theme, _ = db.GetThemePreferences(uid)
	if theme.Theme != "dark" || !theme.WeatherEnabled || theme.Location != "Sydney" {
		t.Errorf("theme round-trip = %+v", theme)
	}

	db.SaveSchedulePreferences(uid, model.SchedulePreferences{Monday: model.StateWorkFromOffice, Friday: model.StateWorkFromHome})
	sched, _ = db.GetSchedulePreferences(uid)
	if sched.Monday != model.StateWorkFromOffice || sched.Friday != model.StateWorkFromHome {
		t.Errorf("schedule round-trip = %+v", sched)
	}

	db.SaveCalendarPreferences(uid, model.CalendarPreferences{TrackingYearStartMonth: 7})
	cal, _ = db.GetCalendarPreferences(uid)
	if cal.TrackingYearStartMonth != 7 {
		t.Errorf("calendar round-trip = %d, want 7", cal.TrackingYearStartMonth)
	}
	// Out-of-range normalises to default on save.
	db.SaveCalendarPreferences(uid, model.CalendarPreferences{TrackingYearStartMonth: 99})
	cal, _ = db.GetCalendarPreferences(uid)
	if cal.TrackingYearStartMonth != 10 {
		t.Errorf("calendar out-of-range = %d, want 10", cal.TrackingYearStartMonth)
	}

	// Attendance target: no target before any save.
	target, err := db.GetTargetPreferences(uid)
	if err != nil || target.TargetPercent != 0 {
		t.Errorf("default target = %+v, err %v, want 0", target, err)
	}
	db.SaveTargetPreferences(uid, model.TargetPreferences{TargetPercent: 50})
	target, _ = db.GetTargetPreferences(uid)
	if target.TargetPercent != 50 {
		t.Errorf("target round-trip = %d, want 50", target.TargetPercent)
	}
	// Out-of-range clamps on save.
	db.SaveTargetPreferences(uid, model.TargetPreferences{TargetPercent: 150})
	target, _ = db.GetTargetPreferences(uid)
	if target.TargetPercent != 100 {
		t.Errorf("target out-of-range = %d, want 100", target.TargetPercent)
	}
	// Zero clears the target.
	db.SaveTargetPreferences(uid, model.TargetPreferences{TargetPercent: 0})
	target, _ = db.GetTargetPreferences(uid)
	if target.TargetPercent != 0 {
		t.Errorf("cleared target = %d, want 0", target.TargetPercent)
	}
}

// Secrets/tokens: save, list active, look up by value, revoke.
func TestPostgresSecretsAndTokens(t *testing.T) {
	db := pgTestDB(t)
	uid := seedUser(t, pgCfg)

	if err := db.SaveSecret(uid, "officetracker:secret-a", "laptop"); err != nil {
		t.Fatalf("SaveSecret: %v", err)
	}
	db.SaveSecret(uid, "officetracker:secret-b", "ci")

	// GetUserBySecret resolves an active secret.
	if id, err := db.GetUserBySecret("officetracker:secret-a"); err != nil || id != uid {
		t.Errorf("GetUserBySecret = (%d, %v), want (%d, nil)", id, err, uid)
	}
	// Unknown secret -> ErrNoUser.
	if _, err := db.GetUserBySecret("nope"); err == nil {
		t.Error("unknown secret should error")
	}

	tokens, err := db.ListActiveTokens(uid)
	if err != nil {
		t.Fatalf("ListActiveTokens: %v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("got %d tokens, want 2", len(tokens))
	}

	// Revoke one token; it disappears from the active list and its secret stops resolving.
	revokeID := tokens[0].TokenID
	if err := db.RevokeToken(uid, revokeID); err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}
	tokens, _ = db.ListActiveTokens(uid)
	if len(tokens) != 1 {
		t.Errorf("after revoke got %d tokens, want 1", len(tokens))
	}

	// Revoking again is a no-op error (row not found / already revoked).
	if err := db.RevokeToken(uid, revokeID); err == nil {
		t.Error("re-revoking should report token not found")
	}

	// RevokeSecretByValue deactivates by secret string.
	if err := db.RevokeSecretByValue("officetracker:secret-b"); err != nil {
		t.Fatalf("RevokeSecretByValue: %v", err)
	}
	if _, err := db.GetUserBySecret("officetracker:secret-b"); err == nil {
		t.Error("revoked secret should no longer resolve")
	}
}

// Auth0 account lifecycle: create user, look up, update profile, link/migrate.
func TestPostgresAuth0Users(t *testing.T) {
	db := pgTestDB(t)

	// Unknown sub -> ErrNoUser.
	if _, err := db.GetUserByAuth0Sub("github|1"); err == nil {
		t.Error("unknown auth0 sub should error")
	}

	// Create a new user by sub.
	uid, err := db.SaveUserByAuth0Sub("google-oauth2|abc", `{"sub":"google-oauth2|abc","nickname":"al"}`)
	if err != nil {
		t.Fatalf("SaveUserByAuth0Sub: %v", err)
	}
	if uid == 0 {
		t.Fatal("expected a non-zero user id")
	}

	got, err := db.GetUserByAuth0Sub("google-oauth2|abc")
	if err != nil || got != uid {
		t.Errorf("GetUserByAuth0Sub = (%d, %v), want (%d, nil)", got, err, uid)
	}

	// Linked accounts reflect the parsed provider and nickname.
	accts, err := db.GetUserLinkedAccounts(uid)
	if err != nil {
		t.Fatalf("GetUserLinkedAccounts: %v", err)
	}
	if len(accts) != 1 || accts[0].Provider != "google-oauth2" || accts[0].Nickname != "al" {
		t.Errorf("linked accounts = %+v", accts)
	}

	// UpdateAuth0Profile changes the stored profile.
	if err := db.UpdateAuth0Profile("google-oauth2|abc", `{"sub":"google-oauth2|abc","nickname":"alice"}`); err != nil {
		t.Fatalf("UpdateAuth0Profile: %v", err)
	}
	accts, _ = db.GetUserLinkedAccounts(uid)
	if accts[0].Nickname != "alice" {
		t.Errorf("updated nickname = %q, want alice", accts[0].Nickname)
	}

	// Linking a second identity to the same user succeeds; linking it to a
	// different user is rejected.
	if err := db.LinkAuth0Account(uid, "github|99", `{"sub":"github|99"}`); err != nil {
		t.Fatalf("LinkAuth0Account: %v", err)
	}
	other := seedUser(t, pgCfg)
	if err := db.LinkAuth0Account(other, "github|99", `{"sub":"github|99"}`); err == nil {
		t.Error("linking an already-owned auth0 account to another user should fail")
	}
}

func TestPostgresSuspension(t *testing.T) {
	db := pgTestDB(t)
	uid := seedUser(t, pgCfg)

	if susp, err := db.IsUserSuspended(uid); err != nil || susp {
		t.Errorf("new user suspended = (%v, %v), want (false, nil)", susp, err)
	}

	// Flip suspension directly and re-check.
	raw, _ := rawConn(pgCfg)
	defer raw.Close()
	if _, err := raw.Exec(`UPDATE users SET suspended = true WHERE user_id = $1;`, uid); err != nil {
		t.Fatalf("set suspended: %v", err)
	}
	if susp, _ := db.IsUserSuspended(uid); !susp {
		t.Error("user should read as suspended")
	}
}

func TestPostgresAggregates(t *testing.T) {
	db := pgTestDB(t)
	uid := seedUser(t, pgCfg)

	db.SaveDay(uid, 1, 1, 2024, model.DayState{State: model.StateWorkFromOffice})
	db.SaveDay(uid, 2, 1, 2024, model.DayState{State: model.StateWorkFromOffice})
	db.SaveDay(uid, 3, 1, 2024, model.DayState{State: model.StateWorkFromHome})
	db.SaveDay(uid, 4, 1, 2024, model.DayState{State: model.StateUntracked}) // excluded

	if n, _ := db.CountTrackedDays(); n != 3 {
		t.Errorf("CountTrackedDays = %d, want 3", n)
	}
	byState, _ := db.CountEntriesByState()
	if byState[model.StateWorkFromOffice] != 2 || byState[model.StateWorkFromHome] != 1 {
		t.Errorf("CountEntriesByState = %v", byState)
	}
	if _, ok := byState[model.StateUntracked]; ok {
		t.Error("untracked should be excluded from CountEntriesByState")
	}
}

// Stats snapshots persist and read back the latest widgets with a timestamp.
func TestPostgresStatsSnapshot(t *testing.T) {
	db := pgTestDB(t)

	// No snapshot yet.
	widgets, ts, err := db.GetLatestStatsSnapshot()
	if err != nil {
		t.Fatalf("GetLatestStatsSnapshot empty: %v", err)
	}
	if widgets != nil || !ts.IsZero() {
		t.Errorf("expected empty snapshot, got %v / %v", widgets, ts)
	}

	want := []model.StatWidget{{Key: "mau", Title: "MAU", Value: "10", Order: 1}}
	if err := db.SaveStatsSnapshot(want); err != nil {
		t.Fatalf("SaveStatsSnapshot: %v", err)
	}
	got, ts, err := db.GetLatestStatsSnapshot()
	if err != nil {
		t.Fatalf("GetLatestStatsSnapshot: %v", err)
	}
	if len(got) != 1 || got[0].Key != "mau" {
		t.Errorf("snapshot widgets = %+v", got)
	}
	if ts.IsZero() || time.Since(ts) > time.Hour {
		t.Errorf("snapshot timestamp looks wrong: %v", ts)
	}
}
