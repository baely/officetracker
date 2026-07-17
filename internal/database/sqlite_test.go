package database

import (
	"path/filepath"
	"testing"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/pkg/model"
)

// newTestDB returns a SQLite Databaser backed by a fresh temp-file database.
// A real file (not :memory:) is used because NewSQLiteClient stats/creates the
// path, and because go-sqlite3's :memory: gives each pooled connection its own
// isolated database.
func newTestDB(t *testing.T) Databaser {
	t.Helper()
	loc := filepath.Join(t.TempDir(), "test.db")
	db, err := NewSQLiteClient(config.SQLite{Location: loc})
	if err != nil {
		t.Fatalf("NewSQLiteClient: %v", err)
	}
	return db
}

func TestSQLiteConstructor(t *testing.T) {
	t.Run("regular file path is created", func(t *testing.T) {
		loc := filepath.Join(t.TempDir(), "new.db")
		if _, err := NewSQLiteClient(config.SQLite{Location: loc}); err != nil {
			t.Fatalf("expected file to be created, got %v", err)
		}
	})

	t.Run("directory location appends default file", func(t *testing.T) {
		dir := t.TempDir()
		db, err := NewSQLiteClient(config.SQLite{Location: dir})
		if err != nil {
			t.Fatalf("directory location: %v", err)
		}
		// A functioning DB proves the schema was created under dir/officetracker.db.
		if err := db.SaveDay(1, 1, 1, 2024, model.DayState{State: model.StateWorkFromOffice}); err != nil {
			t.Fatalf("save into dir-based db: %v", err)
		}
	})

	t.Run("empty location uses default filename in cwd", func(t *testing.T) {
		t.Chdir(t.TempDir())
		db, err := NewSQLiteClient(config.SQLite{Location: ""})
		if err != nil {
			t.Fatalf("empty location: %v", err)
		}
		if err := db.SaveDay(1, 1, 1, 2024, model.DayState{State: model.StateWorkFromHome}); err != nil {
			t.Fatalf("save into default db: %v", err)
		}
	})
}

func TestSQLiteDayRoundTripAndUpsert(t *testing.T) {
	db := newTestDB(t)

	// Missing day reads back as untracked, without error.
	got, err := db.GetDay(1, 5, 3, 2024)
	if err != nil {
		t.Fatalf("GetDay missing: %v", err)
	}
	if got.State != model.StateUntracked {
		t.Errorf("missing day = %d, want untracked", got.State)
	}

	// Save then read.
	if err := db.SaveDay(1, 5, 3, 2024, model.DayState{State: model.StateWorkFromOffice}); err != nil {
		t.Fatalf("SaveDay: %v", err)
	}
	got, _ = db.GetDay(1, 5, 3, 2024)
	if got.State != model.StateWorkFromOffice {
		t.Errorf("saved day = %d, want office", got.State)
	}

	// Re-save the same (day,month,year): INSERT OR REPLACE upserts, no duplicate.
	if err := db.SaveDay(1, 5, 3, 2024, model.DayState{State: model.StateWorkFromHome}); err != nil {
		t.Fatalf("SaveDay upsert: %v", err)
	}
	got, _ = db.GetDay(1, 5, 3, 2024)
	if got.State != model.StateWorkFromHome {
		t.Errorf("upserted day = %d, want home", got.State)
	}
	if n, _ := db.CountTrackedDays(); n != 1 {
		t.Errorf("upsert created a duplicate row: CountTrackedDays = %d, want 1", n)
	}
}

func TestSQLiteMonthRoundTrip(t *testing.T) {
	db := newTestDB(t)
	month := model.MonthState{Days: map[int]model.DayState{
		1:  {State: model.StateWorkFromOffice},
		2:  {State: model.StateWorkFromHome},
		15: {State: model.StateOther},
	}}
	if err := db.SaveMonth(1, 6, 2024, month); err != nil {
		t.Fatalf("SaveMonth: %v", err)
	}

	got, err := db.GetMonth(1, 6, 2024)
	if err != nil {
		t.Fatalf("GetMonth: %v", err)
	}
	if len(got.Days) != 3 {
		t.Fatalf("GetMonth returned %d days, want 3", len(got.Days))
	}
	for day, want := range month.Days {
		if got.Days[day].State != want.State {
			t.Errorf("day %d = %d, want %d", day, got.Days[day].State, want.State)
		}
	}

	// A month with no data returns an empty (non-nil) map.
	empty, _ := db.GetMonth(1, 7, 2024)
	if empty.Days == nil {
		t.Error("GetMonth on empty month returned nil Days map")
	}
	if len(empty.Days) != 0 {
		t.Errorf("empty month has %d days", len(empty.Days))
	}
}

// GetYear applies the tracking-year window: for an October start, tracking year
// 2024 spans Oct 2023 through Sep 2024. Entries outside that window must be
// excluded even though they share a calendar year with included ones.
func TestSQLiteGetYearTrackingWindow(t *testing.T) {
	db := newTestDB(t)

	// (day, month, year) seed spanning the boundary.
	seed := []struct{ month, year int }{
		{9, 2023},  // Sep 2023 - before the window
		{10, 2023}, // Oct 2023 - first month of tracking year 2024
		{12, 2023}, // Dec 2023 - in window
		{1, 2024},  // Jan 2024 - in window
		{9, 2024},  // Sep 2024 - last month of tracking year 2024
		{10, 2024}, // Oct 2024 - next tracking year, excluded
	}
	for _, s := range seed {
		if err := db.SaveDay(1, 1, s.month, s.year, model.DayState{State: model.StateWorkFromOffice}); err != nil {
			t.Fatalf("seed %v: %v", s, err)
		}
	}

	year, err := db.GetYear(1, 2024, 10)
	if err != nil {
		t.Fatalf("GetYear: %v", err)
	}

	wantMonths := map[int]bool{10: true, 12: true, 1: true, 9: true}
	if len(year.Months) != len(wantMonths) {
		t.Errorf("got months %v, want keys %v", keysOf(year.Months), wantMonths)
	}
	for m := range wantMonths {
		if _, ok := year.Months[m]; !ok {
			t.Errorf("expected month %d in tracking year, missing", m)
		}
	}
	// Sep 2023 (month 9, before the window) must not appear via calendar 2023.
	// Month 9 only appears because of Sep 2024, which is in the window.
	if _, ok := year.Months[9]; !ok {
		t.Error("Sep 2024 should be in window")
	}
}

// With a January start the tracking year equals the calendar year.
func TestSQLiteGetYearJanuaryStart(t *testing.T) {
	db := newTestDB(t)
	db.SaveDay(1, 1, 1, 2024, model.DayState{State: model.StateWorkFromOffice})
	db.SaveDay(1, 1, 12, 2024, model.DayState{State: model.StateWorkFromHome})
	db.SaveDay(1, 1, 12, 2023, model.DayState{State: model.StateOther}) // prior calendar year

	year, err := db.GetYear(1, 2024, 1)
	if err != nil {
		t.Fatalf("GetYear: %v", err)
	}
	if _, ok := year.Months[1]; !ok {
		t.Error("Jan 2024 missing")
	}
	if _, ok := year.Months[12]; !ok {
		t.Error("Dec 2024 missing")
	}
	// Dec 2023 must not leak in.
	if ms, ok := year.Months[12]; ok {
		if ms.Days[1].State == model.StateOther {
			t.Error("Dec 2023 entry leaked into calendar year 2024")
		}
	}
}

func TestSQLiteNotes(t *testing.T) {
	db := newTestDB(t)

	// Missing note -> empty, no error.
	n, err := db.GetNote(1, 3, 2024)
	if err != nil || n.Note != "" {
		t.Fatalf("missing note = %q err %v", n.Note, err)
	}

	if err := db.SaveNote(1, 3, 2024, "quarter end crunch"); err != nil {
		t.Fatalf("SaveNote: %v", err)
	}
	n, _ = db.GetNote(1, 3, 2024)
	if n.Note != "quarter end crunch" {
		t.Errorf("note = %q", n.Note)
	}

	// Upsert overwrites.
	db.SaveNote(1, 3, 2024, "updated")
	n, _ = db.GetNote(1, 3, 2024)
	if n.Note != "updated" {
		t.Errorf("upserted note = %q, want updated", n.Note)
	}
}

func TestSQLiteGetNotesWindow(t *testing.T) {
	db := newTestDB(t)
	db.SaveNote(1, 9, 2023, "sep-2023")  // excluded (before Oct start)
	db.SaveNote(1, 10, 2023, "oct-2023") // included
	db.SaveNote(1, 2, 2024, "feb-2024")  // included
	db.SaveNote(1, 11, 2024, "nov-2024") // excluded (next year)

	notes, err := db.GetNotes(1, 2024, 10)
	if err != nil {
		t.Fatalf("GetNotes: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("got %d notes %v, want 2 (oct-2023, feb-2024)", len(notes), notes)
	}
	if notes[10].Note != "oct-2023" || notes[2].Note != "feb-2024" {
		t.Errorf("wrong notes in window: %v", notes)
	}
}

func TestSQLiteThemePreferences(t *testing.T) {
	db := newTestDB(t)

	// Default before anything is saved.
	prefs, err := db.GetThemePreferences(1)
	if err != nil {
		t.Fatalf("GetThemePreferences default: %v", err)
	}
	if prefs.Theme != "default" || prefs.WeatherEnabled || prefs.TimeBasedEnabled {
		t.Errorf("default theme prefs = %+v", prefs)
	}

	saved := model.ThemePreferences{Theme: "dark", WeatherEnabled: true, TimeBasedEnabled: true, Location: "Sydney"}
	if err := db.SaveThemePreferences(1, saved); err != nil {
		t.Fatalf("SaveThemePreferences: %v", err)
	}
	got, _ := db.GetThemePreferences(1)
	if got != saved {
		t.Errorf("theme prefs round-trip = %+v, want %+v", got, saved)
	}

	// Update overwrites (no second row).
	db.SaveThemePreferences(1, model.ThemePreferences{Theme: "light"})
	got, _ = db.GetThemePreferences(1)
	if got.Theme != "light" || got.WeatherEnabled {
		t.Errorf("updated theme prefs = %+v", got)
	}
}

func TestSQLiteSchedulePreferences(t *testing.T) {
	db := newTestDB(t)

	// Default: everything untracked.
	prefs, err := db.GetSchedulePreferences(1)
	if err != nil {
		t.Fatalf("GetSchedulePreferences default: %v", err)
	}
	if prefs.Monday != model.StateUntracked || prefs.Sunday != model.StateUntracked {
		t.Errorf("default schedule prefs not untracked: %+v", prefs)
	}

	saved := model.SchedulePreferences{
		Monday:    model.StateWorkFromOffice,
		Wednesday: model.StateWorkFromHome,
		Friday:    model.StateWorkFromOffice,
	}
	if err := db.SaveSchedulePreferences(1, saved); err != nil {
		t.Fatalf("SaveSchedulePreferences: %v", err)
	}
	got, _ := db.GetSchedulePreferences(1)
	if got != saved {
		t.Errorf("schedule round-trip = %+v, want %+v", got, saved)
	}
}

func TestSQLiteCalendarPreferences(t *testing.T) {
	db := newTestDB(t)

	// Default is the October start month.
	prefs, err := db.GetCalendarPreferences(1)
	if err != nil {
		t.Fatalf("GetCalendarPreferences default: %v", err)
	}
	if prefs.TrackingYearStartMonth != model.DefaultTrackingYearStartMonth {
		t.Errorf("default start month = %d, want %d", prefs.TrackingYearStartMonth, model.DefaultTrackingYearStartMonth)
	}

	if err := db.SaveCalendarPreferences(1, model.CalendarPreferences{TrackingYearStartMonth: 7}); err != nil {
		t.Fatalf("SaveCalendarPreferences: %v", err)
	}
	got, _ := db.GetCalendarPreferences(1)
	if got.TrackingYearStartMonth != 7 {
		t.Errorf("start month = %d, want 7", got.TrackingYearStartMonth)
	}

	// An out-of-range value is normalised to the default on save.
	if err := db.SaveCalendarPreferences(1, model.CalendarPreferences{TrackingYearStartMonth: 99}); err != nil {
		t.Fatalf("SaveCalendarPreferences invalid: %v", err)
	}
	got, _ = db.GetCalendarPreferences(1)
	if got.TrackingYearStartMonth != 10 {
		t.Errorf("out-of-range start month normalised to %d, want 10", got.TrackingYearStartMonth)
	}
}

func TestSQLiteTargetPreferences(t *testing.T) {
	db := newTestDB(t)

	// Defaults to the standard mandate before any save.
	prefs, err := db.GetTargetPreferences(1)
	if err != nil {
		t.Fatalf("GetTargetPreferences default: %v", err)
	}
	if prefs.DefaultTargetPercent != model.DefaultTargetPercent {
		t.Errorf("default target = %d, want %d", prefs.DefaultTargetPercent, model.DefaultTargetPercent)
	}

	if err := db.SaveTargetPreferences(1, model.TargetPreferences{DefaultTargetPercent: 50}); err != nil {
		t.Fatalf("SaveTargetPreferences: %v", err)
	}
	got, _ := db.GetTargetPreferences(1)
	if got.DefaultTargetPercent != 50 {
		t.Errorf("target round-trip = %d, want 50", got.DefaultTargetPercent)
	}

	// An out-of-range value is clamped on save.
	if err := db.SaveTargetPreferences(1, model.TargetPreferences{DefaultTargetPercent: 150}); err != nil {
		t.Fatalf("SaveTargetPreferences invalid: %v", err)
	}
	got, _ = db.GetTargetPreferences(1)
	if got.DefaultTargetPercent != 100 {
		t.Errorf("out-of-range target clamped to %d, want 100", got.DefaultTargetPercent)
	}

	// Zero clears the target.
	if err := db.SaveTargetPreferences(1, model.TargetPreferences{DefaultTargetPercent: 0}); err != nil {
		t.Fatalf("SaveTargetPreferences clear: %v", err)
	}
	got, _ = db.GetTargetPreferences(1)
	if got.DefaultTargetPercent != 0 {
		t.Errorf("cleared target = %d, want 0", got.DefaultTargetPercent)
	}
}

// CountTrackedDays and CountEntriesByState both exclude untracked entries and
// feed the public stats dashboard.
func TestSQLiteAggregates(t *testing.T) {
	db := newTestDB(t)
	db.SaveDay(1, 1, 1, 2024, model.DayState{State: model.StateWorkFromOffice})
	db.SaveDay(1, 2, 1, 2024, model.DayState{State: model.StateWorkFromOffice})
	db.SaveDay(1, 3, 1, 2024, model.DayState{State: model.StateWorkFromHome})
	db.SaveDay(1, 4, 1, 2024, model.DayState{State: model.StateOther})
	db.SaveDay(1, 5, 1, 2024, model.DayState{State: model.StateUntracked}) // excluded

	n, err := db.CountTrackedDays()
	if err != nil {
		t.Fatalf("CountTrackedDays: %v", err)
	}
	if n != 4 {
		t.Errorf("CountTrackedDays = %d, want 4 (untracked excluded)", n)
	}

	byState, err := db.CountEntriesByState()
	if err != nil {
		t.Fatalf("CountEntriesByState: %v", err)
	}
	if byState[model.StateWorkFromOffice] != 2 {
		t.Errorf("office count = %d, want 2", byState[model.StateWorkFromOffice])
	}
	if byState[model.StateWorkFromHome] != 1 {
		t.Errorf("home count = %d, want 1", byState[model.StateWorkFromHome])
	}
	if byState[model.StateOther] != 1 {
		t.Errorf("other count = %d, want 1", byState[model.StateOther])
	}
	if _, ok := byState[model.StateUntracked]; ok {
		t.Error("untracked should not appear in CountEntriesByState")
	}
}

// The standalone SQLite backend stubs out the multi-user / Auth0 / token / stats
// features. Lock in those contracts so callers can rely on them.
func TestSQLiteStandaloneStubs(t *testing.T) {
	db := newTestDB(t)

	if id, err := db.GetUserByGHID("anything"); id != 1 || err != nil {
		t.Errorf("GetUserByGHID = (%d,%v), want (1,nil)", id, err)
	}
	if id, err := db.GetUserBySecret("anything"); id != 1 || err != nil {
		t.Errorf("GetUserBySecret = (%d,%v), want (1,nil)", id, err)
	}
	if accts, err := db.GetUserLinkedAccounts(1); err != nil || len(accts) != 0 {
		t.Errorf("GetUserLinkedAccounts = (%v,%v), want (empty,nil)", accts, err)
	}
	if _, err := db.GetUserByAuth0Sub("sub"); err == nil {
		t.Error("GetUserByAuth0Sub should error in standalone mode")
	}
	if _, err := db.SaveUserByAuth0Sub("sub", "{}"); err == nil {
		t.Error("SaveUserByAuth0Sub should error in standalone mode")
	}
	if err := db.LinkAuth0Account(1, "sub", "{}"); err == nil {
		t.Error("LinkAuth0Account should error in standalone mode")
	}
	if susp, err := db.IsUserSuspended(1); susp || err != nil {
		t.Errorf("IsUserSuspended = (%v,%v), want (false,nil)", susp, err)
	}
	if err := db.SaveSecret(1, "s", "n"); err != nil {
		t.Errorf("SaveSecret stub err = %v", err)
	}
	if toks, err := db.ListActiveTokens(1); err != nil || len(toks) != 0 {
		t.Errorf("ListActiveTokens = (%v,%v), want (empty,nil)", toks, err)
	}
	if err := db.SaveStatsSnapshot(nil); err != nil {
		t.Errorf("SaveStatsSnapshot stub err = %v", err)
	}
	widgets, ts, err := db.GetLatestStatsSnapshot()
	if err != nil || widgets != nil || !ts.IsZero() {
		t.Errorf("GetLatestStatsSnapshot = (%v,%v,%v), want (nil, zero, nil)", widgets, ts, err)
	}
}

func keysOf(m map[int]model.MonthState) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
