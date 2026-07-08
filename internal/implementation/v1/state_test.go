package v1

import (
	"errors"
	"testing"

	"github.com/baely/officetracker/internal/database/dbtest"
	"github.com/baely/officetracker/pkg/model"
)

var errInjected = errors.New("injected failure")

// mergeScheduleWithYear overlays a user's weekly schedule onto their actual
// recorded attendance: for days with no real data (or explicitly untracked
// data) it fills in the corresponding "scheduled" state, while real data is
// left untouched.
func TestMergeScheduleWithYear(t *testing.T) {
	svc := &Service{} // merge touches no dependencies

	// January 2024 (a January start keeps every month in calendar year 2024).
	// Jan 1, 8, 15 are Mondays; Jan 3 a Wednesday; Jan 5 a Friday; Jan 2 a Tuesday.
	year := model.YearState{Months: map[int]model.MonthState{
		1: {Days: map[int]model.DayState{
			8:  {State: model.StateWorkFromOffice}, // real data - must be preserved
			15: {State: model.StateUntracked},      // explicit untracked - eligible for overlay
		}},
	}}
	sched := model.SchedulePreferences{
		Monday:    model.StateWorkFromOffice,
		Wednesday: model.StateWorkFromHome,
		Friday:    model.StateOther,
		Tuesday:   model.StateUntracked, // untracked schedule contributes nothing
	}

	got := svc.mergeScheduleWithYear(year, sched, 2024, 1)
	days := got.Months[1].Days

	checks := []struct {
		day  int
		want model.State
	}{
		{1, model.StateScheduledWorkFromOffice},  // Mon, empty -> scheduled office
		{3, model.StateScheduledWorkFromHome},    // Wed, empty -> scheduled home
		{5, model.StateScheduledOther},           // Fri, empty -> scheduled other
		{8, model.StateWorkFromOffice},           // Mon, real office -> preserved
		{15, model.StateScheduledWorkFromOffice}, // Mon, untracked -> overlaid
	}
	for _, c := range checks {
		if days[c.day].State != c.want {
			t.Errorf("Jan %d = %d, want %d", c.day, days[c.day].State, c.want)
		}
	}

	// Tuesday has an untracked schedule, so Jan 2 must not be materialised.
	if _, ok := days[2]; ok {
		t.Errorf("Jan 2 (untracked schedule) should not be added, got %+v", days[2])
	}
}

// The merge must cope with a completely empty year (nil Months map) and still
// produce scheduled overlays.
func TestMergeScheduleWithYearEmptyInput(t *testing.T) {
	svc := &Service{}
	sched := model.SchedulePreferences{Monday: model.StateWorkFromHome}

	got := svc.mergeScheduleWithYear(model.YearState{}, sched, 2024, 1)
	if got.Months == nil {
		t.Fatal("merge did not initialise Months map")
	}
	// Jan 1 2024 is a Monday.
	if got.Months[1].Days[1].State != model.StateScheduledWorkFromHome {
		t.Errorf("Jan 1 = %d, want scheduled WFH", got.Months[1].Days[1].State)
	}
}

func TestGetDayPutDay(t *testing.T) {
	db := dbtest.New()
	svc := &Service{db: db}

	// Save then read back.
	_, err := svc.PutDay(model.PutDayRequest{
		Meta: model.PutDayRequestMeta{UserID: 1, Year: 2024, Month: 3, Day: 5},
		Data: model.DayState{State: model.StateWorkFromOffice},
	})
	if err != nil {
		t.Fatalf("PutDay: %v", err)
	}

	resp, err := svc.GetDay(model.GetDayRequest{
		Meta: model.GetDayRequestMeta{UserID: 1, Year: 2024, Month: 3, Day: 5},
	})
	if err != nil {
		t.Fatalf("GetDay: %v", err)
	}
	if resp.Data.State != model.StateWorkFromOffice {
		t.Errorf("GetDay state = %d, want office", resp.Data.State)
	}
}

func TestGetDayError(t *testing.T) {
	db := dbtest.New()
	db.Errs = map[string]error{"GetDay": errInjected}
	svc := &Service{db: db}
	if _, err := svc.GetDay(model.GetDayRequest{}); err == nil {
		t.Fatal("expected GetDay to propagate db error")
	}
}

func TestGetMonthPutMonth(t *testing.T) {
	db := dbtest.New()
	svc := &Service{db: db}

	_, err := svc.PutMonth(model.PutMonthRequest{
		Meta: model.PutMonthRequestMeta{UserID: 1, Year: 2024, Month: 4},
		Data: model.MonthState{Days: map[int]model.DayState{
			1: {State: model.StateWorkFromHome},
			2: {State: model.StateWorkFromOffice},
		}},
	})
	if err != nil {
		t.Fatalf("PutMonth: %v", err)
	}

	resp, err := svc.GetMonth(model.GetMonthRequest{
		Meta: model.GetMonthRequestMeta{UserID: 1, Year: 2024, Month: 4},
	})
	if err != nil {
		t.Fatalf("GetMonth: %v", err)
	}
	if len(resp.Data.Days) != 2 || resp.Data.Days[2].State != model.StateWorkFromOffice {
		t.Errorf("GetMonth data = %+v", resp.Data)
	}
}

// GetYear resolves the tracking start month, fetches the year, and overlays the
// schedule — a day with a schedule but no data comes back as a scheduled state.
func TestGetYearMergesSchedule(t *testing.T) {
	db := dbtest.New()
	db.SaveCalendarPreferences(1, model.CalendarPreferences{TrackingYearStartMonth: 1})
	db.SaveSchedulePreferences(1, model.SchedulePreferences{Monday: model.StateWorkFromOffice})
	// A real WFH day on a Wednesday (Jan 3 2024).
	db.SaveDay(1, 3, 1, 2024, model.DayState{State: model.StateWorkFromHome})
	svc := &Service{db: db}

	resp, err := svc.GetYear(model.GetYearRequest{
		Meta: model.GetYearRequestMeta{UserID: 1, Year: 2024},
	})
	if err != nil {
		t.Fatalf("GetYear: %v", err)
	}
	jan := resp.Data.Months[1].Days
	if jan[1].State != model.StateScheduledWorkFromOffice {
		t.Errorf("Jan 1 (scheduled Monday) = %d, want scheduled office", jan[1].State)
	}
	if jan[3].State != model.StateWorkFromHome {
		t.Errorf("Jan 3 (real WFH) = %d, want WFH preserved", jan[3].State)
	}
}

func TestGetYearCalendarPrefsError(t *testing.T) {
	db := dbtest.New()
	db.Errs = map[string]error{"GetCalendarPreferences": errInjected}
	svc := &Service{db: db}
	if _, err := svc.GetYear(model.GetYearRequest{}); err == nil {
		t.Fatal("expected GetYear to fail when calendar prefs cannot be read")
	}
}

func TestNotesRoundTrip(t *testing.T) {
	db := dbtest.New()
	svc := &Service{db: db}

	_, err := svc.PutNote(model.PutNoteRequest{
		Meta: model.PutNoteRequestMeta{UserID: 1, Year: 2024, Month: 6},
		Data: model.Note{Note: "eofy"},
	})
	if err != nil {
		t.Fatalf("PutNote: %v", err)
	}

	one, err := svc.GetNote(model.GetNoteRequest{
		Meta: model.GetNoteRequestMeta{UserID: 1, Year: 2024, Month: 6},
	})
	if err != nil {
		t.Fatalf("GetNote: %v", err)
	}
	if one.Data.Note != "eofy" {
		t.Errorf("GetNote = %q, want eofy", one.Data.Note)
	}

	all, err := svc.GetNotes(model.GetNotesRequest{
		Meta: model.GetNotesRequestMeta{UserID: 1, Year: 2024},
	})
	if err != nil {
		t.Fatalf("GetNotes: %v", err)
	}
	// June falls inside tracking year 2024 for the default October start.
	if all.Data[6].Note != "eofy" {
		t.Errorf("GetNotes[6] = %q, want eofy", all.Data[6].Note)
	}
}
