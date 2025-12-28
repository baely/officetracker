package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateConstants(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		expected int
	}{
		{"StateUntracked", StateUntracked, 0},
		{"StateWorkFromHome", StateWorkFromHome, 1},
		{"StateWorkFromOffice", StateWorkFromOffice, 2},
		{"StateOther", StateOther, 3},
		{"StateScheduledWorkFromHome", StateScheduledWorkFromHome, 4},
		{"StateScheduledWorkFromOffice", StateScheduledWorkFromOffice, 5},
		{"StateScheduledOther", StateScheduledOther, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, int(tt.state))
		})
	}
}

func TestDayState_JSON(t *testing.T) {
	tests := []struct {
		name     string
		dayState DayState
		wantJSON string
	}{
		{
			name:     "untracked state",
			dayState: DayState{State: StateUntracked},
			wantJSON: `{"state":0}`,
		},
		{
			name:     "work from home state",
			dayState: DayState{State: StateWorkFromHome},
			wantJSON: `{"state":1}`,
		},
		{
			name:     "work from office state",
			dayState: DayState{State: StateWorkFromOffice},
			wantJSON: `{"state":2}`,
		},
		{
			name:     "other state",
			dayState: DayState{State: StateOther},
			wantJSON: `{"state":3}`,
		},
		{
			name:     "scheduled work from home",
			dayState: DayState{State: StateScheduledWorkFromHome},
			wantJSON: `{"state":4}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			got, err := json.Marshal(tt.dayState)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(got))

			// Test unmarshaling
			var dayState DayState
			err = json.Unmarshal([]byte(tt.wantJSON), &dayState)
			require.NoError(t, err)
			assert.Equal(t, tt.dayState, dayState)
		})
	}
}

func TestMonthState_JSON(t *testing.T) {
	tests := []struct {
		name       string
		monthState MonthState
		wantJSON   string
	}{
		{
			name: "empty month",
			monthState: MonthState{
				Days: map[int]DayState{},
			},
			wantJSON: `{"days":{}}`,
		},
		{
			name: "month with single day",
			monthState: MonthState{
				Days: map[int]DayState{
					1: {State: StateWorkFromHome},
				},
			},
			wantJSON: `{"days":{"1":{"state":1}}}`,
		},
		{
			name: "month with multiple days",
			monthState: MonthState{
				Days: map[int]DayState{
					1:  {State: StateWorkFromHome},
					2:  {State: StateWorkFromOffice},
					15: {State: StateOther},
				},
			},
			wantJSON: `{"days":{"1":{"state":1},"2":{"state":2},"15":{"state":3}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			got, err := json.Marshal(tt.monthState)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(got))

			// Test unmarshaling
			var monthState MonthState
			err = json.Unmarshal([]byte(tt.wantJSON), &monthState)
			require.NoError(t, err)
			assert.Equal(t, len(tt.monthState.Days), len(monthState.Days))
			for day, expectedState := range tt.monthState.Days {
				actualState, ok := monthState.Days[day]
				assert.True(t, ok, "Missing day %d", day)
				assert.Equal(t, expectedState, actualState)
			}
		})
	}
}

func TestYearState_JSON(t *testing.T) {
	tests := []struct {
		name      string
		yearState YearState
		wantJSON  string
	}{
		{
			name: "empty year",
			yearState: YearState{
				Months: map[int]MonthState{},
			},
			wantJSON: `{"months":{}}`,
		},
		{
			name: "year with single month",
			yearState: YearState{
				Months: map[int]MonthState{
					1: {
						Days: map[int]DayState{
							1: {State: StateWorkFromHome},
						},
					},
				},
			},
			wantJSON: `{"months":{"1":{"days":{"1":{"state":1}}}}}`,
		},
		{
			name: "year with multiple months",
			yearState: YearState{
				Months: map[int]MonthState{
					1: {
						Days: map[int]DayState{
							1: {State: StateWorkFromHome},
						},
					},
					12: {
						Days: map[int]DayState{
							25: {State: StateOther},
						},
					},
				},
			},
			wantJSON: `{"months":{"1":{"days":{"1":{"state":1}}},"12":{"days":{"25":{"state":3}}}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			got, err := json.Marshal(tt.yearState)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(got))

			// Test unmarshaling
			var yearState YearState
			err = json.Unmarshal([]byte(tt.wantJSON), &yearState)
			require.NoError(t, err)
			assert.Equal(t, len(tt.yearState.Months), len(yearState.Months))
		})
	}
}

func TestNote_JSON(t *testing.T) {
	tests := []struct {
		name     string
		note     Note
		wantJSON string
	}{
		{
			name:     "empty note",
			note:     Note{Note: ""},
			wantJSON: `{"note":""}`,
		},
		{
			name:     "simple note",
			note:     Note{Note: "This is a test note"},
			wantJSON: `{"note":"This is a test note"}`,
		},
		{
			name:     "note with special characters",
			note:     Note{Note: "Note with \"quotes\" and\nnewlines"},
			wantJSON: `{"note":"Note with \"quotes\" and\nnewlines"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			got, err := json.Marshal(tt.note)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(got))

			// Test unmarshaling
			var note Note
			err = json.Unmarshal([]byte(tt.wantJSON), &note)
			require.NoError(t, err)
			assert.Equal(t, tt.note, note)
		})
	}
}

func TestThemePreferences_JSON(t *testing.T) {
	tests := []struct {
		name     string
		theme    ThemePreferences
		wantJSON string
	}{
		{
			name: "empty theme preferences",
			theme: ThemePreferences{
				Theme:            "",
				WeatherEnabled:   false,
				TimeBasedEnabled: false,
				Location:         "",
			},
			wantJSON: `{"theme":"","weather_enabled":false,"time_based_enabled":false}`,
		},
		{
			name: "full theme preferences",
			theme: ThemePreferences{
				Theme:            "dark",
				WeatherEnabled:   true,
				TimeBasedEnabled: true,
				Location:         "Sydney",
			},
			wantJSON: `{"theme":"dark","weather_enabled":true,"time_based_enabled":true,"location":"Sydney"}`,
		},
		{
			name: "theme without location",
			theme: ThemePreferences{
				Theme:            "light",
				WeatherEnabled:   false,
				TimeBasedEnabled: true,
			},
			wantJSON: `{"theme":"light","weather_enabled":false,"time_based_enabled":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			got, err := json.Marshal(tt.theme)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(got))

			// Test unmarshaling
			var theme ThemePreferences
			err = json.Unmarshal([]byte(tt.wantJSON), &theme)
			require.NoError(t, err)
			assert.Equal(t, tt.theme.Theme, theme.Theme)
			assert.Equal(t, tt.theme.WeatherEnabled, theme.WeatherEnabled)
			assert.Equal(t, tt.theme.TimeBasedEnabled, theme.TimeBasedEnabled)
			if tt.theme.Location != "" {
				assert.Equal(t, tt.theme.Location, theme.Location)
			}
		})
	}
}

func TestSchedulePreferences_JSON(t *testing.T) {
	tests := []struct {
		name     string
		schedule SchedulePreferences
		wantJSON string
	}{
		{
			name: "empty schedule",
			schedule: SchedulePreferences{
				Monday:    StateUntracked,
				Tuesday:   StateUntracked,
				Wednesday: StateUntracked,
				Thursday:  StateUntracked,
				Friday:    StateUntracked,
				Saturday:  StateUntracked,
				Sunday:    StateUntracked,
			},
			wantJSON: `{"monday":0,"tuesday":0,"wednesday":0,"thursday":0,"friday":0,"saturday":0,"sunday":0}`,
		},
		{
			name: "full week WFH",
			schedule: SchedulePreferences{
				Monday:    StateWorkFromHome,
				Tuesday:   StateWorkFromHome,
				Wednesday: StateWorkFromHome,
				Thursday:  StateWorkFromHome,
				Friday:    StateWorkFromHome,
				Saturday:  StateUntracked,
				Sunday:    StateUntracked,
			},
			wantJSON: `{"monday":1,"tuesday":1,"wednesday":1,"thursday":1,"friday":1,"saturday":0,"sunday":0}`,
		},
		{
			name: "hybrid schedule",
			schedule: SchedulePreferences{
				Monday:    StateWorkFromOffice,
				Tuesday:   StateWorkFromHome,
				Wednesday: StateWorkFromOffice,
				Thursday:  StateWorkFromHome,
				Friday:    StateWorkFromOffice,
				Saturday:  StateUntracked,
				Sunday:    StateUntracked,
			},
			wantJSON: `{"monday":2,"tuesday":1,"wednesday":2,"thursday":1,"friday":2,"saturday":0,"sunday":0}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			got, err := json.Marshal(tt.schedule)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(got))

			// Test unmarshaling
			var schedule SchedulePreferences
			err = json.Unmarshal([]byte(tt.wantJSON), &schedule)
			require.NoError(t, err)
			assert.Equal(t, tt.schedule, schedule)
		})
	}
}

func TestLinkedAccount_JSON(t *testing.T) {
	tests := []struct {
		name     string
		account  LinkedAccount
		wantJSON string
	}{
		{
			name: "github account",
			account: LinkedAccount{
				Provider:        "github",
				ProviderDisplay: "GitHub",
				Nickname:        "johndoe",
			},
			wantJSON: `{"provider":"github","provider_display":"GitHub","nickname":"johndoe"}`,
		},
		{
			name: "google account",
			account: LinkedAccount{
				Provider:        "google-oauth2",
				ProviderDisplay: "Google",
				Nickname:        "jane.doe",
			},
			wantJSON: `{"provider":"google-oauth2","provider_display":"Google","nickname":"jane.doe"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			got, err := json.Marshal(tt.account)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(got))

			// Test unmarshaling
			var account LinkedAccount
			err = json.Unmarshal([]byte(tt.wantJSON), &account)
			require.NoError(t, err)
			assert.Equal(t, tt.account, account)
		})
	}
}
