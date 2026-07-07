package model

import (
	"encoding/json"
	"strings"
	"testing"
)

// The State enum values are a stable wire/storage contract: they are persisted
// as integers in the database and sent to the frontend as JSON numbers. If any
// of these change, existing stored data is silently reinterpreted.
func TestStateEnumValues(t *testing.T) {
	cases := []struct {
		state State
		want  int
	}{
		{StateUntracked, 0},
		{StateWorkFromHome, 1},
		{StateWorkFromOffice, 2},
		{StateOther, 3},
		{StateScheduledWorkFromHome, 4},
		{StateScheduledWorkFromOffice, 5},
		{StateScheduledOther, 6},
	}
	for _, c := range cases {
		if int(c.state) != c.want {
			t.Errorf("state %v = %d, want %d", c.state, int(c.state), c.want)
		}
	}
}

func TestDefaultTrackingYearStartMonth(t *testing.T) {
	if DefaultTrackingYearStartMonth != 10 {
		t.Fatalf("DefaultTrackingYearStartMonth = %d, want 10 (October)", DefaultTrackingYearStartMonth)
	}
}

// DayState serialises its State as a bare JSON integer.
func TestDayStateJSON(t *testing.T) {
	b, err := json.Marshal(DayState{State: StateWorkFromOffice})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(b); got != `{"state":2}` {
		t.Fatalf("DayState JSON = %s, want {\"state\":2}", got)
	}

	var back DayState
	if err := json.Unmarshal([]byte(`{"state":1}`), &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.State != StateWorkFromHome {
		t.Fatalf("round-trip state = %d, want %d", back.State, StateWorkFromHome)
	}
}

// A full YearState -> JSON -> YearState round-trip must preserve the nested
// month/day/state structure exactly (this is what the API returns for /state/{year}).
func TestYearStateRoundTrip(t *testing.T) {
	orig := YearState{
		Months: map[int]MonthState{
			10: {Days: map[int]DayState{
				1: {State: StateWorkFromOffice},
				2: {State: StateWorkFromHome},
			}},
			11: {Days: map[int]DayState{
				15: {State: StateOther},
			}},
		},
	}

	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var back YearState
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for month, ms := range orig.Months {
		for day, ds := range ms.Days {
			if back.Months[month].Days[day].State != ds.State {
				t.Errorf("month %d day %d: got %d want %d", month, day,
					back.Months[month].Days[day].State, ds.State)
			}
		}
	}
}

func TestNoteJSON(t *testing.T) {
	b, err := json.Marshal(Note{Note: "wfh due to strike"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(b); got != `{"note":"wfh due to strike"}` {
		t.Fatalf("Note JSON = %s", got)
	}
}

// SchedulePreferences must use lowercase weekday JSON keys with States as ints —
// the settings page reads these directly.
func TestSchedulePreferencesJSON(t *testing.T) {
	b, err := json.Marshal(SchedulePreferences{
		Monday:    StateWorkFromOffice,
		Wednesday: StateWorkFromHome,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	for _, want := range []string{`"monday":2`, `"tuesday":0`, `"wednesday":1`, `"sunday":0`} {
		if !contains(got, want) {
			t.Errorf("schedule JSON missing %q in %s", want, got)
		}
	}
}

// Location and the stats optional fields are omitempty; empty values must not
// appear in the marshalled output.
func TestOmitemptyFields(t *testing.T) {
	tb, _ := json.Marshal(ThemePreferences{Theme: "dark"})
	if contains(string(tb), "location") {
		t.Errorf("empty Location should be omitted, got %s", tb)
	}
	tb2, _ := json.Marshal(ThemePreferences{Theme: "dark", Location: "Sydney"})
	if !contains(string(tb2), `"location":"Sydney"`) {
		t.Errorf("non-empty Location should be present, got %s", tb2)
	}

	wb, _ := json.Marshal(StatWidget{Key: "mau", Title: "MAU", Value: "10", Order: 1})
	s := string(wb)
	for _, absent := range []string{"prefix", "unit", "group"} {
		if contains(s, absent) {
			t.Errorf("empty %s should be omitted, got %s", absent, s)
		}
	}
	// order:0 is NOT omitempty for widgets; it must always render.
	wb0, _ := json.Marshal(StatWidget{Key: "x"})
	if !contains(string(wb0), `"order":0`) {
		t.Errorf("order must always render, got %s", wb0)
	}
}

// GetStatsResponse omits ComputedAt when empty (no snapshot yet).
func TestGetStatsResponseComputedAtOmitempty(t *testing.T) {
	empty, _ := json.Marshal(GetStatsResponse{Widgets: []StatWidget{}})
	if contains(string(empty), "computedAt") {
		t.Errorf("empty ComputedAt should be omitted, got %s", empty)
	}
	set, _ := json.Marshal(GetStatsResponse{ComputedAt: "2026-07-07T00:00:00Z"})
	if !contains(string(set), "computedAt") {
		t.Errorf("set ComputedAt should be present, got %s", set)
	}
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
