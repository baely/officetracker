package dbtest

import (
	"errors"
	"testing"
	"time"

	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/pkg/model"
)

// The fake is depended on by report/v1/auth tests, so its behaviour must mirror
// the real database semantics it stands in for. These tests lock that in.

func TestFakeDefaults(t *testing.T) {
	f := New()
	if theme, _ := f.GetThemePreferences(1); theme.Theme != "default" {
		t.Errorf("default theme = %q, want default", theme.Theme)
	}
	if cal, _ := f.GetCalendarPreferences(1); cal.TrackingYearStartMonth != model.DefaultTrackingYearStartMonth {
		t.Errorf("default calendar = %d, want %d", cal.TrackingYearStartMonth, model.DefaultTrackingYearStartMonth)
	}
	if day, _ := f.GetDay(1, 1, 1, 2024); day.State != model.StateUntracked {
		t.Errorf("missing day = %d, want untracked", day.State)
	}
}

func TestFakeDayAndMonthRoundTrip(t *testing.T) {
	f := New()
	f.SaveDay(1, 5, 3, 2024, model.DayState{State: model.StateWorkFromOffice})
	if d, _ := f.GetDay(1, 5, 3, 2024); d.State != model.StateWorkFromOffice {
		t.Errorf("day = %d, want office", d.State)
	}

	f.SaveMonth(1, 4, 2024, model.MonthState{Days: map[int]model.DayState{
		1: {State: model.StateWorkFromHome},
		2: {State: model.StateWorkFromOffice},
	}})
	m, _ := f.GetMonth(1, 4, 2024)
	if len(m.Days) != 2 || m.Days[2].State != model.StateWorkFromOffice {
		t.Errorf("month = %+v", m.Days)
	}
}

// GetYear applies the same tracking-year window as the real backends.
func TestFakeGetYearWindow(t *testing.T) {
	f := New()
	for _, s := range []struct{ month, year int }{{9, 2023}, {10, 2023}, {1, 2024}, {9, 2024}, {10, 2024}} {
		f.SaveDay(1, 1, s.month, s.year, model.DayState{State: model.StateWorkFromOffice})
	}
	year, _ := f.GetYear(1, 2024, 10)
	if len(year.Months) != 3 {
		t.Fatalf("got months %v, want 3 (10,1,9)", year.Months)
	}
	for _, m := range []int{10, 1, 9} {
		if _, ok := year.Months[m]; !ok {
			t.Errorf("month %d missing", m)
		}
	}
}

func TestFakeNotesWindow(t *testing.T) {
	f := New()
	f.SaveNote(1, 9, 2023, "before")
	f.SaveNote(1, 10, 2023, "oct")
	f.SaveNote(1, 2, 2024, "feb")
	f.SaveNote(1, 11, 2024, "after")
	notes, _ := f.GetNotes(1, 2024, 10)
	if len(notes) != 2 || notes[10].Note != "oct" || notes[2].Note != "feb" {
		t.Errorf("notes window = %v", notes)
	}
}

func TestFakeAggregatesExcludeUntracked(t *testing.T) {
	f := New()
	f.SaveDay(1, 1, 1, 2024, model.DayState{State: model.StateWorkFromOffice})
	f.SaveDay(1, 2, 1, 2024, model.DayState{State: model.StateWorkFromHome})
	f.SaveDay(1, 3, 1, 2024, model.DayState{State: model.StateUntracked})

	if n, _ := f.CountTrackedDays(); n != 2 {
		t.Errorf("CountTrackedDays = %d, want 2", n)
	}
	byState, _ := f.CountEntriesByState()
	if byState[model.StateWorkFromOffice] != 1 || byState[model.StateWorkFromHome] != 1 {
		t.Errorf("byState = %v", byState)
	}
	if _, ok := byState[model.StateUntracked]; ok {
		t.Error("untracked should be excluded")
	}
}

func TestFakeErrorInjection(t *testing.T) {
	f := New()
	sentinel := errors.New("boom")
	f.Errs = map[string]error{"GetDay": sentinel}
	if _, err := f.GetDay(1, 1, 1, 2024); !errors.Is(err, sentinel) {
		t.Errorf("GetDay error = %v, want sentinel", err)
	}
	// Un-injected methods still work.
	if err := f.SaveDay(1, 1, 1, 2024, model.DayState{}); err != nil {
		t.Errorf("SaveDay should not error: %v", err)
	}
}

func TestFakeRecordingAndHooks(t *testing.T) {
	f := New()

	f.SaveSecret(7, "officetracker:abc", "laptop")
	if len(f.SavedSecrets) != 1 || f.SavedSecrets[0].UserID != 7 || f.SavedSecrets[0].Name != "laptop" {
		t.Errorf("SavedSecrets = %+v", f.SavedSecrets)
	}

	f.RevokeToken(7, 3)
	if len(f.RevokedTokens) != 1 || f.RevokedTokens[0].TokenID != 3 {
		t.Errorf("RevokedTokens = %+v", f.RevokedTokens)
	}

	f.LinkAuth0Account(7, "github|1", "{}")
	if len(f.LinkedAuth0) != 1 || f.LinkedAuth0[0].Sub != "github|1" {
		t.Errorf("LinkedAuth0 = %+v", f.LinkedAuth0)
	}

	// Default user hooks return ErrNoUser; overrides take effect.
	if _, err := f.GetUserBySecret("x"); !errors.Is(err, database.ErrNoUser) {
		t.Errorf("default GetUserBySecret error = %v, want ErrNoUser", err)
	}
	f.GetUserBySecretFn = func(string) (int, error) { return 99, nil }
	if id, _ := f.GetUserBySecret("x"); id != 99 {
		t.Errorf("hooked GetUserBySecret = %d, want 99", id)
	}
}

func TestFakeStatsSnapshot(t *testing.T) {
	f := New()
	ts := time.Date(2026, 7, 7, 0, 0, 0, 0, time.UTC)
	f.Snapshot = []model.StatWidget{{Key: "mau"}}
	f.SetStatsTime(ts)

	got, gotTS, err := f.GetLatestStatsSnapshot()
	if err != nil || len(got) != 1 || !gotTS.Equal(ts) {
		t.Errorf("GetLatestStatsSnapshot = (%v, %v, %v)", got, gotTS, err)
	}

	f.SaveStatsSnapshot([]model.StatWidget{{Key: "x"}})
	if len(f.SavedSnapshots) != 1 {
		t.Errorf("SavedSnapshots = %d, want 1", len(f.SavedSnapshots))
	}
}
