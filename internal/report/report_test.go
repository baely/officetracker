package report

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/baely/officetracker/internal/database/dbtest"
	"github.com/baely/officetracker/pkg/model"
)

var errFake = errors.New("injected failure")

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// getMonths yields the first-of-month for every month in [start, end), after
// truncating start to the first of its month.
func TestGetMonths(t *testing.T) {
	var got []string
	for m := range getMonths(date(2024, 1, 15), date(2024, 4, 1)) {
		got = append(got, m.Format("2006-01"))
	}
	want := []string{"2024-01", "2024-02", "2024-03"}
	if !equalStrings(got, want) {
		t.Fatalf("getMonths = %v, want %v", got, want)
	}

	// Range spanning a year boundary.
	got = nil
	for m := range getMonths(date(2023, 11, 5), date(2024, 2, 1)) {
		got = append(got, m.Format("2006-01"))
	}
	if !equalStrings(got, []string{"2023-11", "2023-12", "2024-01"}) {
		t.Fatalf("year boundary getMonths = %v", got)
	}

	// Empty range yields nothing.
	got = nil
	for m := range getMonths(date(2024, 5, 1), date(2024, 5, 1)) {
		got = append(got, m.Format("2006-01"))
	}
	if len(got) != 0 {
		t.Fatalf("empty range yielded %v", got)
	}
}

// getDays yields only Monday-Friday within [start, end).
func TestGetDaysWeekdaysOnly(t *testing.T) {
	// 2024-01-01 is a Monday; the range through Jan 8 (exclusive) covers one full
	// week, so weekends (Jan 6 Sat, Jan 7 Sun) must be skipped.
	var got []string
	for d := range getDays(date(2024, 1, 1), date(2024, 1, 8)) {
		got = append(got, d.Format("2006-01-02 Mon"))
	}
	want := []string{
		"2024-01-01 Mon", "2024-01-02 Tue", "2024-01-03 Wed",
		"2024-01-04 Thu", "2024-01-05 Fri",
	}
	if !equalStrings(got, want) {
		t.Fatalf("getDays = %v, want %v", got, want)
	}
}

func TestReportGet(t *testing.T) {
	r := Report{Months: map[Key]model.MonthState{
		{Month: time.March, Year: 2024}: {Days: map[int]model.DayState{1: {State: model.StateWorkFromOffice}}},
	}}
	if got := r.Get(time.March, 2024); got.Days[1].State != model.StateWorkFromOffice {
		t.Errorf("Get present month = %+v", got)
	}
	// Absent key returns a zero MonthState (nil Days), not a panic.
	if got := r.Get(time.April, 2024); got.Days != nil {
		t.Errorf("Get absent month = %+v, want zero", got)
	}
}

func TestGetState(t *testing.T) {
	cases := []struct {
		in   model.State
		want string
	}{
		{model.StateWorkFromHome, "Home"},
		{model.StateWorkFromOffice, "Office"},
		{model.StateOther, ""},
		{model.StateUntracked, ""},
		{model.StateScheduledWorkFromHome, ""}, // scheduled states are not mapped
	}
	for _, c := range cases {
		if got := getState(c.in); got != c.want {
			t.Errorf("getState(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestGetStatusString(t *testing.T) {
	if getStatusString(model.StateWorkFromHome) != "Home" {
		t.Error("WFH should map to Home")
	}
	if getStatusString(model.StateWorkFromOffice) != "Office" {
		t.Error("office should map to Office")
	}
	if getStatusString(model.StateOther) != "" || getStatusString(model.StateUntracked) != "" {
		t.Error("other/untracked should map to empty")
	}
}

func TestIsScheduledDay(t *testing.T) {
	prefs := model.SchedulePreferences{
		Monday:  model.StateWorkFromOffice,
		Tuesday: model.StateUntracked,
	}
	if !isScheduledDay(date(2024, 1, 1), prefs) { // Monday
		t.Error("Monday with office schedule should be scheduled")
	}
	if isScheduledDay(date(2024, 1, 2), prefs) { // Tuesday untracked
		t.Error("Tuesday untracked should not be scheduled")
	}
}

func TestBuildCsv(t *testing.T) {
	got := buildCsv([]csvLine{
		{Date: "2024-01-01", State: "Office"},
		{Date: "2024-01-02", State: ""},
	})
	want := "Date,State\n2024-01-01,Office\n2024-01-02,\n"
	if string(got) != want {
		t.Fatalf("buildCsv = %q, want %q", got, want)
	}
}

func TestPadString(t *testing.T) {
	if got := padString("x", 2, 3); got != "  x   " {
		t.Fatalf("padString = %q", got)
	}
	if got := padString("", 0, 0); got != "" {
		t.Fatalf("padString empty = %q", got)
	}
}

// generateSummaries counts office days as present, WFH days toward the total,
// and adds untracked-but-scheduled days to the total. Untracked, unscheduled
// days are ignored entirely.
func TestGenerateSummaries(t *testing.T) {
	report := Report{Months: map[Key]model.MonthState{
		{Month: time.January, Year: 2024}: {Days: map[int]model.DayState{
			2: {State: model.StateWorkFromOffice},
			3: {State: model.StateWorkFromOffice},
			4: {State: model.StateWorkFromHome},
			5: {State: model.StateUntracked},
		}},
	}}
	// No schedule, so no phantom scheduled days.
	p := newPDF(report, model.SchedulePreferences{}, "", date(2024, 1, 1), date(2024, 2, 1))

	if len(p.monthlySummaries) != 1 {
		t.Fatalf("expected 1 monthly summary, got %d", len(p.monthlySummaries))
	}
	for _, s := range p.monthlySummaries {
		if s.Present != 2 {
			t.Errorf("Present = %d, want 2", s.Present)
		}
		if s.Total != 3 {
			t.Errorf("Total = %d, want 3 (2 office + 1 wfh)", s.Total)
		}
		wantPct := float64(2) / float64(3) * 100
		if s.Percent != wantPct {
			t.Errorf("Percent = %v, want %v", s.Percent, wantPct)
		}
	}
}

// countScheduledDays counts weekdays whose schedule is set and whose actual
// state is missing or untracked. Days already recorded as office are excluded
// (they are counted as present elsewhere and must not be double-counted).
func TestCountScheduledDays(t *testing.T) {
	// Monday scheduled as office. January 2024 has Mondays on the 1st, 8th, 15th,
	// 22nd and 29th (5 total). Record the 1st as actual office so it is skipped.
	report := Report{Months: map[Key]model.MonthState{
		{Month: time.January, Year: 2024}: {Days: map[int]model.DayState{
			1: {State: model.StateWorkFromOffice},
		}},
	}}
	prefs := model.SchedulePreferences{Monday: model.StateWorkFromOffice}
	p := newPDF(report, prefs, "", date(2024, 1, 1), date(2024, 2, 1))

	if got := p.countScheduledDays(2024, time.January); got != 4 {
		t.Errorf("countScheduledDays = %d, want 4 (5 Mondays minus the 1 recorded office day)", got)
	}

	// With no schedule at all, nothing is counted.
	p2 := newPDF(Report{Months: map[Key]model.MonthState{}}, model.SchedulePreferences{}, "", date(2024, 1, 1), date(2024, 2, 1))
	if got := p2.countScheduledDays(2024, time.January); got != 0 {
		t.Errorf("countScheduledDays with no schedule = %d, want 0", got)
	}
}

// GenerateCSV produces a header plus one line per weekday with the mapped state.
func TestGenerateCSVIntegration(t *testing.T) {
	db := dbtest.New()
	db.SaveDay(1, 2, 1, 2024, model.DayState{State: model.StateWorkFromOffice}) // Tue
	db.SaveDay(1, 3, 1, 2024, model.DayState{State: model.StateWorkFromHome})   // Wed
	r := New(db)

	out, err := r.GenerateCSV(1, date(2024, 1, 1), date(2024, 1, 8))
	if err != nil {
		t.Fatalf("GenerateCSV: %v", err)
	}
	want := "Date,State\n" +
		"2024-01-01,\n" + // Mon, no data
		"2024-01-02,Office\n" +
		"2024-01-03,Home\n" +
		"2024-01-04,\n" +
		"2024-01-05,\n"
	if string(out) != want {
		t.Fatalf("GenerateCSV =\n%q\nwant\n%q", out, want)
	}
}

// An untracked day that falls on a scheduled weekday is labelled "Scheduled".
func TestGenerateCSVScheduledDay(t *testing.T) {
	db := dbtest.New()
	db.SaveSchedulePreferences(1, model.SchedulePreferences{Thursday: model.StateWorkFromOffice})
	r := New(db)

	out, err := r.GenerateCSV(1, date(2024, 1, 4), date(2024, 1, 5)) // Thursday only
	if err != nil {
		t.Fatalf("GenerateCSV: %v", err)
	}
	want := "Date,State\n2024-01-04,Scheduled\n"
	if string(out) != want {
		t.Fatalf("scheduled-day CSV = %q, want %q", out, want)
	}
}

func TestGenerateCSVScheduleError(t *testing.T) {
	db := dbtest.New()
	db.Errs = map[string]error{"GetSchedulePreferences": errFake}
	r := New(db)
	if _, err := r.GenerateCSV(1, date(2024, 1, 1), date(2024, 1, 8)); err == nil {
		t.Fatal("expected GenerateCSV to propagate schedule-preferences error")
	}
}

// GeneratePDF returns a valid, non-empty PDF document.
func TestGeneratePDFIntegration(t *testing.T) {
	db := dbtest.New()
	db.SaveMonth(1, 1, 2024, model.MonthState{Days: map[int]model.DayState{
		2: {State: model.StateWorkFromOffice},
		3: {State: model.StateWorkFromHome},
	}})
	r := New(db)

	out, err := r.GeneratePDF(1, "Alice", date(2024, 1, 1), date(2024, 2, 1))
	if err != nil {
		t.Fatalf("GeneratePDF: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("GeneratePDF returned no bytes")
	}
	if !bytes.HasPrefix(out, []byte("%PDF")) {
		t.Fatalf("output is not a PDF (prefix = %q)", out[:min(8, len(out))])
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
