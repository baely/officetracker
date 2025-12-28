package v1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/baely/officetracker/pkg/model"
	"github.com/baely/officetracker/testutil/fixtures"
	"github.com/baely/officetracker/testutil/mocks"
)

func TestGetDay(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	// Setup test data
	mockDB.SetDay(1, 15, 10, 2024, model.DayState{State: model.StateWorkFromHome})

	req := model.GetDayRequest{
		Meta: model.GetDayRequestMeta{
			UserID: 1,
			Year:   2024,
			Month:  10,
			Day:    15,
		},
	}

	resp, err := service.GetDay(req)
	require.NoError(t, err)
	assert.Equal(t, model.StateWorkFromHome, resp.Data.State)

	// Verify DB was called
	assert.Len(t, mockDB.GetDayCalls, 1)
	assert.Equal(t, 1, mockDB.GetDayCalls[0].UserID)
	assert.Equal(t, 15, mockDB.GetDayCalls[0].Day)
	assert.Equal(t, 10, mockDB.GetDayCalls[0].Month)
	assert.Equal(t, 2024, mockDB.GetDayCalls[0].Year)
}

func TestGetDay_NotFound(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	req := model.GetDayRequest{
		Meta: model.GetDayRequestMeta{
			UserID: 1,
			Year:   2024,
			Month:  10,
			Day:    15,
		},
	}

	resp, err := service.GetDay(req)
	require.NoError(t, err)
	// Should return untracked state when not found
	assert.Equal(t, model.StateUntracked, resp.Data.State)
}

func TestPutDay(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	req := model.PutDayRequest{
		Meta: model.PutDayRequestMeta{
			UserID: 1,
			Year:   2024,
			Month:  10,
			Day:    15,
		},
		Data: model.DayState{State: model.StateWorkFromOffice},
	}

	_, err := service.PutDay(req)
	require.NoError(t, err)

	// Verify DB was called
	assert.Len(t, mockDB.SaveDayCalls, 1)
	assert.Equal(t, 1, mockDB.SaveDayCalls[0].UserID)
	assert.Equal(t, 15, mockDB.SaveDayCalls[0].Day)
	assert.Equal(t, 10, mockDB.SaveDayCalls[0].Month)
	assert.Equal(t, 2024, mockDB.SaveDayCalls[0].Year)
	assert.Equal(t, model.StateWorkFromOffice, mockDB.SaveDayCalls[0].State.State)
}

func TestGetMonth(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	// Setup test data
	mockDB.SetDay(1, 1, 10, 2024, model.DayState{State: model.StateWorkFromHome})
	mockDB.SetDay(1, 2, 10, 2024, model.DayState{State: model.StateWorkFromOffice})
	mockDB.SetDay(1, 15, 10, 2024, model.DayState{State: model.StateWorkFromHome})

	req := model.GetMonthRequest{
		Meta: model.GetMonthRequestMeta{
			UserID: 1,
			Year:   2024,
			Month:  10,
		},
	}

	resp, err := service.GetMonth(req)
	require.NoError(t, err)
	assert.Len(t, resp.Data.Days, 3)
	assert.Equal(t, model.StateWorkFromHome, resp.Data.Days[1].State)
	assert.Equal(t, model.StateWorkFromOffice, resp.Data.Days[2].State)
	assert.Equal(t, model.StateWorkFromHome, resp.Data.Days[15].State)
}

func TestPutMonth(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	monthState := model.MonthState{
		Days: map[int]model.DayState{
			1:  {State: model.StateWorkFromHome},
			2:  {State: model.StateWorkFromOffice},
			15: {State: model.StateWorkFromHome},
		},
	}

	req := model.PutMonthRequest{
		Meta: model.PutMonthRequestMeta{
			UserID: 1,
			Year:   2024,
			Month:  10,
		},
		Data: monthState,
	}

	_, err := service.PutMonth(req)
	require.NoError(t, err)

	// Verify DB was called
	assert.Len(t, mockDB.SaveMonthCalls, 1)
	assert.Equal(t, 1, mockDB.SaveMonthCalls[0].UserID)
	assert.Equal(t, 10, mockDB.SaveMonthCalls[0].Month)
	assert.Equal(t, 2024, mockDB.SaveMonthCalls[0].Year)
}

func TestGetYear_WithScheduleMerging(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	// Setup actual state data in 2024 calendar year
	// January 1, 2024 is a Monday - set it to Office
	mockDB.SetDay(1, 1, 1, 2024, model.DayState{State: model.StateWorkFromOffice})

	// Setup schedule preferences (Mondays WFH)
	schedulePrefs := fixtures.NewSchedulePreferences().
		WithWeekday(time.Monday, model.StateWorkFromHome).
		Build()
	mockDB.SetSchedulePreferences(1, schedulePrefs)

	req := model.GetYearRequest{
		Meta: model.GetYearRequestMeta{
			UserID: 1,
			Year:   2024,
		},
	}

	resp, err := service.GetYear(req)
	require.NoError(t, err)

	// The actual state (Office) should override the schedule (WFH)
	jan := resp.Data.Months[1]
	assert.Equal(t, model.StateWorkFromOffice, jan.Days[1].State,
		"Actual state should override schedule")
}

func TestMergeScheduleWithYear_EmptyYearWithSchedule(t *testing.T) {
	service := New(mocks.NewMockDB(), nil)

	yearState := model.YearState{Months: make(map[int]model.MonthState)}
	schedulePrefs := fixtures.NewSchedulePreferences().
		WithWeekday(time.Monday, model.StateWorkFromHome).
		Build()

	result := service.mergeScheduleWithYear(yearState, schedulePrefs, 2024)

	// Should have all 12 months
	assert.Equal(t, 12, len(result.Months))

	// Check that Mondays in October 2023 have scheduled WFH state
	oct2023 := result.Months[10]

	// Oct 2, 9, 16, 23, 30, 2023 are Mondays
	expectedMondays := []int{2, 9, 16, 23, 30}
	for _, day := range expectedMondays {
		dayState, exists := oct2023.Days[day]
		assert.True(t, exists, "Monday %d should exist", day)
		assert.Equal(t, model.StateScheduledWorkFromHome, dayState.State,
			"Monday Oct %d should be scheduled WFH", day)
	}
}

func TestMergeScheduleWithYear_ActualStateOverridesSchedule(t *testing.T) {
	service := New(mocks.NewMockDB(), nil)

	// Year with actual state on a Monday
	yearState := fixtures.NewYearState(2024).
		WithDay(10, 7, model.StateWorkFromOffice). // Monday Oct 7, 2024 - actual office
		Build()

	// Schedule says Mondays are WFH
	schedulePrefs := fixtures.NewSchedulePreferences().
		WithWeekday(time.Monday, model.StateWorkFromHome).
		Build()

	result := service.mergeScheduleWithYear(yearState, schedulePrefs, 2024)

	// Actual state should NOT be overridden
	oct := result.Months[10]
	assert.Equal(t, model.StateWorkFromOffice, oct.Days[7].State,
		"Actual state should not be overridden by schedule")
}

func TestMergeScheduleWithYear_UntrackedShowsSchedule(t *testing.T) {
	service := New(mocks.NewMockDB(), nil)

	// Year with explicitly untracked state
	// January 1, 2024 is a Monday
	yearState := fixtures.NewYearState(2024).
		WithDay(1, 1, model.StateUntracked). // Monday Jan 1, 2024 - explicitly untracked
		Build()

	// Schedule says Mondays are WFH
	schedulePrefs := fixtures.NewSchedulePreferences().
		WithWeekday(time.Monday, model.StateWorkFromHome).
		Build()

	result := service.mergeScheduleWithYear(yearState, schedulePrefs, 2024)

	// Untracked should show schedule
	jan := result.Months[1]
	assert.Equal(t, model.StateScheduledWorkFromHome, jan.Days[1].State,
		"Untracked state should show schedule")
}

func TestMergeScheduleWithYear_AcademicYearBoundaries(t *testing.T) {
	service := New(mocks.NewMockDB(), nil)

	yearState := model.YearState{Months: make(map[int]model.MonthState)}
	schedulePrefs := fixtures.NewSchedulePreferences().
		WithWorkdaysWFH(). // Mon-Fri WFH
		Build()

	result := service.mergeScheduleWithYear(yearState, schedulePrefs, 2024)

	// Verify October is from year-1 (2023)
	// October 1, 2023 is a Sunday
	// October 2, 2023 is a Monday (should have scheduled state)
	oct := result.Months[10]
	assert.Equal(t, model.StateScheduledWorkFromHome, oct.Days[2].State,
		"October 2 (Monday) should be scheduled WFH")

	// Verify September is from year (2024)
	// September 2, 2024 is a Monday (should have scheduled state)
	sep := result.Months[9]
	assert.Equal(t, model.StateScheduledWorkFromHome, sep.Days[2].State,
		"September 2 (Monday) should be scheduled WFH")
}

func TestMergeScheduleWithYear_LeapYear(t *testing.T) {
	service := New(mocks.NewMockDB(), nil)

	yearState := model.YearState{Months: make(map[int]model.MonthState)}

	// Feb 29, 2024 is a Thursday
	schedulePrefs := fixtures.NewSchedulePreferences().
		WithWeekday(time.Thursday, model.StateWorkFromOffice).
		Build()

	result := service.mergeScheduleWithYear(yearState, schedulePrefs, 2024)

	// Verify February has 29 days (leap year)
	feb := result.Months[2]
	dayState, exists := feb.Days[29]
	assert.True(t, exists, "February 29 should exist in leap year 2024")
	assert.Equal(t, model.StateScheduledWorkFromOffice, dayState.State,
		"February 29 (Thursday) should be scheduled office")
}

func TestMergeScheduleWithYear_NonLeapYear(t *testing.T) {
	service := New(mocks.NewMockDB(), nil)

	yearState := model.YearState{Months: make(map[int]model.MonthState)}
	schedulePrefs := fixtures.NewSchedulePreferences().
		WithWorkdaysOffice().
		Build()

	result := service.mergeScheduleWithYear(yearState, schedulePrefs, 2023)

	// Verify February has only 28 days (non-leap year)
	feb := result.Months[2]
	_, day29Exists := feb.Days[29]
	assert.False(t, day29Exists, "February 29 should not exist in non-leap year 2023")

	_, day28Exists := feb.Days[28]
	assert.True(t, day28Exists, "February 28 should exist")
}

func TestMergeScheduleWithYear_MixedSchedule(t *testing.T) {
	service := New(mocks.NewMockDB(), nil)

	yearState := model.YearState{Months: make(map[int]model.MonthState)}

	// Hybrid schedule: Mon/Wed/Fri office, Tue/Thu home
	schedulePrefs := model.SchedulePreferences{
		Monday:    model.StateWorkFromOffice,
		Tuesday:   model.StateWorkFromHome,
		Wednesday: model.StateWorkFromOffice,
		Thursday:  model.StateWorkFromHome,
		Friday:    model.StateWorkFromOffice,
		Saturday:  model.StateUntracked,
		Sunday:    model.StateUntracked,
	}

	result := service.mergeScheduleWithYear(yearState, schedulePrefs, 2024)

	// January 2024: Mon 1, Tue 2, Wed 3, Thu 4, Fri 5
	jan := result.Months[1]
	assert.Equal(t, model.StateScheduledWorkFromOffice, jan.Days[1].State, "Mon should be office")
	assert.Equal(t, model.StateScheduledWorkFromHome, jan.Days[2].State, "Tue should be home")
	assert.Equal(t, model.StateScheduledWorkFromOffice, jan.Days[3].State, "Wed should be office")
	assert.Equal(t, model.StateScheduledWorkFromHome, jan.Days[4].State, "Thu should be home")
	assert.Equal(t, model.StateScheduledWorkFromOffice, jan.Days[5].State, "Fri should be office")
}

func TestGetNote(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	err := mockDB.SaveNote(1, 10, 2024, "Test note")
	require.NoError(t, err)

	req := model.GetNoteRequest{
		Meta: model.GetNoteRequestMeta{
			UserID: 1,
			Year:   2024,
			Month:  10,
		},
	}

	resp, err := service.GetNote(req)
	require.NoError(t, err)
	assert.Equal(t, "Test note", resp.Data.Note)
}

func TestPutNote(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	req := model.PutNoteRequest{
		Meta: model.PutNoteRequestMeta{
			UserID: 1,
			Year:   2024,
			Month:  10,
		},
		Data: model.Note{Note: "My test note"},
	}

	_, err := service.PutNote(req)
	require.NoError(t, err)

	// Verify note was saved
	assert.Len(t, mockDB.SaveNoteCalls, 1)
	assert.Equal(t, "My test note", mockDB.SaveNoteCalls[0].Note)
}

func TestGetNotes(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	// Save multiple notes
	mockDB.SaveNote(1, 1, 2024, "January note")
	mockDB.SaveNote(1, 6, 2024, "June note")
	mockDB.SaveNote(1, 12, 2024, "December note")

	req := model.GetNotesRequest{
		Meta: model.GetNotesRequestMeta{
			UserID: 1,
			Year:   2024,
		},
	}

	resp, err := service.GetNotes(req)
	require.NoError(t, err)
	assert.Len(t, resp.Data, 3)
	assert.Equal(t, "January note", resp.Data[1].Note)
	assert.Equal(t, "June note", resp.Data[6].Note)
	assert.Equal(t, "December note", resp.Data[12].Note)
}

// Error handling tests for state functions
func TestGetDay_ErrorHandling(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnGetDay = assert.AnError
	service := New(mockDB, nil)

	req := model.GetDayRequest{
		Meta: model.GetDayRequestMeta{
			UserID: 1,
			Year:   2024,
			Month:  10,
			Day:    15,
		},
	}

	_, err := service.GetDay(req)
	assert.Error(t, err)
}

func TestPutDay_ErrorHandling(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnSaveDay = assert.AnError
	service := New(mockDB, nil)

	req := model.PutDayRequest{
		Meta: model.PutDayRequestMeta{
			UserID: 1,
			Year:   2024,
			Month:  10,
			Day:    15,
		},
		Data: model.DayState{State: model.StateWorkFromHome},
	}

	_, err := service.PutDay(req)
	assert.Error(t, err)
}

func TestGetMonth_ErrorHandling(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnGetMonth = assert.AnError
	service := New(mockDB, nil)

	req := model.GetMonthRequest{
		Meta: model.GetMonthRequestMeta{
			UserID: 1,
			Year:   2024,
			Month:  10,
		},
	}

	_, err := service.GetMonth(req)
	assert.Error(t, err)
}

func TestPutMonth_ErrorHandling(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnSaveMonth = assert.AnError
	service := New(mockDB, nil)

	req := model.PutMonthRequest{
		Meta: model.PutMonthRequestMeta{
			UserID: 1,
			Year:   2024,
			Month:  10,
		},
		Data: model.MonthState{
			Days: map[int]model.DayState{
				1: {State: model.StateWorkFromHome},
			},
		},
	}

	_, err := service.PutMonth(req)
	assert.Error(t, err)
}

func TestGetYear_ErrorHandling(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnGetYear = assert.AnError
	service := New(mockDB, nil)

	req := model.GetYearRequest{
		Meta: model.GetYearRequestMeta{
			UserID: 1,
			Year:   2024,
		},
	}

	_, err := service.GetYear(req)
	assert.Error(t, err)
}

func TestGetYear_SchedulePrefsError(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnGetSchedulePreferences = assert.AnError
	service := New(mockDB, nil)

	req := model.GetYearRequest{
		Meta: model.GetYearRequestMeta{
			UserID: 1,
			Year:   2024,
		},
	}

	_, err := service.GetYear(req)
	assert.Error(t, err)
}

func TestGetNote_ErrorHandling(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnGetNote = assert.AnError
	service := New(mockDB, nil)

	req := model.GetNoteRequest{
		Meta: model.GetNoteRequestMeta{
			UserID: 1,
			Year:   2024,
			Month:  10,
		},
	}

	_, err := service.GetNote(req)
	assert.Error(t, err)
}

func TestPutNote_ErrorHandling(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnSaveNote = assert.AnError
	service := New(mockDB, nil)

	req := model.PutNoteRequest{
		Meta: model.PutNoteRequestMeta{
			UserID: 1,
			Year:   2024,
			Month:  10,
		},
		Data: model.Note{Note: "Test note"},
	}

	_, err := service.PutNote(req)
	assert.Error(t, err)
}

func TestGetNotes_ErrorHandling(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnGetNotes = assert.AnError
	service := New(mockDB, nil)

	req := model.GetNotesRequest{
		Meta: model.GetNotesRequestMeta{
			UserID: 1,
			Year:   2024,
		},
	}

	_, err := service.GetNotes(req)
	assert.Error(t, err)
}

func TestGetSettings_LinkedAccountsError(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnGetUserLinkedAccounts = assert.AnError
	service := New(mockDB, nil)

	req := model.GetSettingsRequest{
		Meta: model.GetSettingsRequestMeta{
			UserID: 1,
		},
	}

	_, err := service.GetSettings(req)
	assert.Error(t, err)
}
