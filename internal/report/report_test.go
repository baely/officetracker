package report

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/baely/officetracker/pkg/model"
	"github.com/baely/officetracker/testutil/mocks"
)

func TestReport_Get(t *testing.T) {
	report := Report{
		Months: map[Key]model.MonthState{
			{Month: time.October, Year: 2024}: {
				Days: map[int]model.DayState{
					15: {State: model.StateWorkFromHome},
				},
			},
		},
	}

	monthData := report.Get(time.October, 2024)
	assert.Equal(t, model.StateWorkFromHome, monthData.Days[15].State)

	// Non-existent month returns empty state
	emptyMonth := report.Get(time.January, 2024)
	assert.Empty(t, emptyMonth.Days)
}

func TestGenerate(t *testing.T) {
	mockDB := mocks.NewMockDB()
	reporter := New(mockDB)

	// Setup test data for Oct-Dec 2024
	mockDB.SetDay(1, 15, 10, 2024, model.DayState{State: model.StateWorkFromHome})
	mockDB.SetDay(1, 20, 11, 2024, model.DayState{State: model.StateWorkFromOffice})
	mockDB.SetDay(1, 5, 12, 2024, model.DayState{State: model.StateWorkFromHome})

	start := time.Date(2024, 10, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	report, err := reporter.Generate(1, start, end)

	require.NoError(t, err)
	assert.Len(t, report.Months, 3) // Oct, Nov, Dec

	// Check October data
	octData := report.Get(time.October, 2024)
	assert.Equal(t, model.StateWorkFromHome, octData.Days[15].State)

	// Check November data
	novData := report.Get(time.November, 2024)
	assert.Equal(t, model.StateWorkFromOffice, novData.Days[20].State)

	// Check December data
	decData := report.Get(time.December, 2024)
	assert.Equal(t, model.StateWorkFromHome, decData.Days[5].State)
}

func TestGenerate_ErrorHandling(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnGetMonth = assert.AnError
	reporter := New(mockDB)

	start := time.Date(2024, 10, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC)

	_, err := reporter.Generate(1, start, end)
	assert.Error(t, err)
}

func TestGetMonths(t *testing.T) {
	start := time.Date(2024, 10, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	var months []time.Time
	for month := range getMonths(start, end) {
		months = append(months, month)
	}

	require.Len(t, months, 4) // Oct, Nov, Dec, Jan
	assert.Equal(t, time.October, months[0].Month())
	assert.Equal(t, 2024, months[0].Year())
	assert.Equal(t, time.November, months[1].Month())
	assert.Equal(t, time.December, months[2].Month())
	assert.Equal(t, time.January, months[3].Month())
	assert.Equal(t, 2025, months[3].Year())

	// All months should start on day 1
	for _, month := range months {
		assert.Equal(t, 1, month.Day())
	}
}

func TestGetMonths_SingleMonth(t *testing.T) {
	start := time.Date(2024, 10, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC)

	var months []time.Time
	for month := range getMonths(start, end) {
		months = append(months, month)
	}

	require.Len(t, months, 1) // Only Oct
	assert.Equal(t, time.October, months[0].Month())
}

func TestGetDays(t *testing.T) {
	// November 2024 - starts on Friday (Nov 1)
	start := time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 11, 8, 0, 0, 0, 0, time.UTC)

	var days []time.Time
	for day := range getDays(start, end) {
		days = append(days, day)
	}

	// Nov 1 (Fri), 4 (Mon), 5 (Tue), 6 (Wed), 7 (Thu) - 5 weekdays
	require.Len(t, days, 5)

	// Verify no weekends
	for _, day := range days {
		assert.NotEqual(t, time.Saturday, day.Weekday())
		assert.NotEqual(t, time.Sunday, day.Weekday())
	}
}

func TestGetDays_WeekendSkipping(t *testing.T) {
	// Start on Saturday
	start := time.Date(2024, 11, 2, 0, 0, 0, 0, time.UTC) // Saturday
	end := time.Date(2024, 11, 5, 0, 0, 0, 0, time.UTC)   // Tuesday

	var days []time.Time
	for day := range getDays(start, end) {
		days = append(days, day)
	}

	// Should only get Monday (Nov 4) - skips Sat, Sun
	require.Len(t, days, 1)
	assert.Equal(t, time.Monday, days[0].Weekday())
	assert.Equal(t, 4, days[0].Day())
}

func TestGetState(t *testing.T) {
	tests := []struct {
		name     string
		state    model.State
		expected string
	}{
		{
			name:     "Work from home",
			state:    model.StateWorkFromHome,
			expected: "Home",
		},
		{
			name:     "Work from office",
			state:    model.StateWorkFromOffice,
			expected: "Office",
		},
		{
			name:     "Untracked",
			state:    model.StateUntracked,
			expected: "",
		},
		{
			name:     "Other",
			state:    model.StateOther,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getState(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsScheduledDay(t *testing.T) {
	schedulePrefs := model.SchedulePreferences{
		Monday:    model.StateWorkFromOffice,
		Tuesday:   model.StateWorkFromHome,
		Wednesday: model.StateUntracked,
		Thursday:  model.StateWorkFromOffice,
		Friday:    model.StateWorkFromHome,
		Saturday:  model.StateUntracked,
		Sunday:    model.StateUntracked,
	}

	tests := []struct {
		name       string
		day        time.Time
		isScheduled bool
	}{
		{
			name:        "Monday is scheduled",
			day:         time.Date(2024, 11, 4, 0, 0, 0, 0, time.UTC), // Monday
			isScheduled: true,
		},
		{
			name:        "Tuesday is scheduled",
			day:         time.Date(2024, 11, 5, 0, 0, 0, 0, time.UTC), // Tuesday
			isScheduled: true,
		},
		{
			name:        "Wednesday is not scheduled",
			day:         time.Date(2024, 11, 6, 0, 0, 0, 0, time.UTC), // Wednesday
			isScheduled: false,
		},
		{
			name:        "Saturday is not scheduled",
			day:         time.Date(2024, 11, 2, 0, 0, 0, 0, time.UTC), // Saturday
			isScheduled: false,
		},
		{
			name:        "Sunday is not scheduled",
			day:         time.Date(2024, 11, 3, 0, 0, 0, 0, time.UTC), // Sunday
			isScheduled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isScheduledDay(tt.day, schedulePrefs)
			assert.Equal(t, tt.isScheduled, result)
		})
	}
}

func TestGenerateCSV(t *testing.T) {
	mockDB := mocks.NewMockDB()
	reporter := New(mockDB)

	// Setup test data
	mockDB.SetDay(1, 4, 11, 2024, model.DayState{State: model.StateWorkFromOffice}) // Monday
	mockDB.SetDay(1, 5, 11, 2024, model.DayState{State: model.StateWorkFromHome})   // Tuesday
	// Nov 6 (Wed) is untracked

	// Setup schedule preferences
	schedulePrefs := model.SchedulePreferences{
		Monday:    model.StateWorkFromOffice,
		Tuesday:   model.StateWorkFromHome,
		Wednesday: model.StateWorkFromOffice, // Scheduled but untracked
		Thursday:  model.StateUntracked,
		Friday:    model.StateUntracked,
	}
	mockDB.SetSchedulePreferences(1, schedulePrefs)

	start := time.Date(2024, 11, 4, 0, 0, 0, 0, time.UTC) // Monday
	end := time.Date(2024, 11, 7, 0, 0, 0, 0, time.UTC)   // Thursday

	csvBytes, err := reporter.GenerateCSV(1, start, end)

	require.NoError(t, err)
	csvString := string(csvBytes)

	// Check CSV header
	assert.Contains(t, csvString, "Date,State")

	// Check specific entries
	assert.Contains(t, csvString, "2024-11-04,Office")     // Monday - actual office
	assert.Contains(t, csvString, "2024-11-05,Home")       // Tuesday - actual home
	assert.Contains(t, csvString, "2024-11-06,Scheduled")  // Wednesday - scheduled but untracked

	// Verify CSV format (should have 4 lines: header + 3 weekdays)
	lines := strings.Split(strings.TrimSpace(csvString), "\n")
	assert.Len(t, lines, 4)
}

func TestGenerateCSV_ErrorOnGenerate(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnGetMonth = assert.AnError
	reporter := New(mockDB)

	start := time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)

	_, err := reporter.GenerateCSV(1, start, end)
	assert.Error(t, err)
}

func TestGenerateCSV_ErrorOnSchedulePrefs(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnGetSchedulePreferences = assert.AnError
	reporter := New(mockDB)

	start := time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)

	_, err := reporter.GenerateCSV(1, start, end)
	assert.Error(t, err)
}

func TestBuildCsv(t *testing.T) {
	lines := []csvLine{
		{Date: "2024-11-01", State: "Office"},
		{Date: "2024-11-02", State: "Home"},
		{Date: "2024-11-03", State: ""},
	}

	csv := buildCsv(lines)
	csvString := string(csv)

	assert.Contains(t, csvString, "Date,State")
	assert.Contains(t, csvString, "2024-11-01,Office")
	assert.Contains(t, csvString, "2024-11-02,Home")
	assert.Contains(t, csvString, "2024-11-03,")
}

func TestGeneratePDF(t *testing.T) {
	mockDB := mocks.NewMockDB()
	reporter := New(mockDB)

	// Setup test data
	mockDB.SetDay(1, 4, 11, 2024, model.DayState{State: model.StateWorkFromOffice})
	mockDB.SetDay(1, 5, 11, 2024, model.DayState{State: model.StateWorkFromHome})

	schedulePrefs := model.SchedulePreferences{
		Monday:  model.StateWorkFromOffice,
		Tuesday: model.StateWorkFromHome,
	}
	mockDB.SetSchedulePreferences(1, schedulePrefs)

	start := time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)

	pdfBytes, err := reporter.GeneratePDF(1, "Test User", start, end)

	require.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)
	// PDF files start with %PDF-
	assert.Equal(t, "%PDF-", string(pdfBytes[:5]))
}

func TestGeneratePDF_ErrorOnGenerate(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnGetMonth = assert.AnError
	reporter := New(mockDB)

	start := time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)

	_, err := reporter.GeneratePDF(1, "Test User", start, end)
	assert.Error(t, err)
}

func TestGeneratePDF_ErrorOnSchedulePrefs(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnGetSchedulePreferences = assert.AnError
	reporter := New(mockDB)

	start := time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)

	_, err := reporter.GeneratePDF(1, "Test User", start, end)
	assert.Error(t, err)
}

func TestGetStatusString(t *testing.T) {
	tests := []struct {
		name     string
		state    model.State
		expected string
	}{
		{
			name:     "Work from home",
			state:    model.StateWorkFromHome,
			expected: "Home",
		},
		{
			name:     "Work from office",
			state:    model.StateWorkFromOffice,
			expected: "Office",
		},
		{
			name:     "Untracked",
			state:    model.StateUntracked,
			expected: "",
		},
		{
			name:     "Other",
			state:    model.StateOther,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStatusString(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPadString(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		paddingLeft  int
		paddingRight int
		expected     string
	}{
		{
			name:         "Both sides",
			input:        "test",
			paddingLeft:  2,
			paddingRight: 3,
			expected:     "  test   ",
		},
		{
			name:         "Left only",
			input:        "test",
			paddingLeft:  3,
			paddingRight: 0,
			expected:     "   test",
		},
		{
			name:         "Right only",
			input:        "test",
			paddingLeft:  0,
			paddingRight: 4,
			expected:     "test    ",
		},
		{
			name:         "No padding",
			input:        "test",
			paddingLeft:  0,
			paddingRight: 0,
			expected:     "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padString(tt.input, tt.paddingLeft, tt.paddingRight)
			assert.Equal(t, tt.expected, result)
		})
	}
}
