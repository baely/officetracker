package assert

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/baely/officetracker/pkg/model"
)

// AssertDayState compares two DayState structs
func AssertDayState(t *testing.T, expected, actual model.DayState, msgAndArgs ...interface{}) bool {
	t.Helper()
	return assert.Equal(t, expected.State, actual.State, msgAndArgs...)
}

// AssertMonthState compares two MonthState structs
func AssertMonthState(t *testing.T, expected, actual model.MonthState, msgAndArgs ...interface{}) bool {
	t.Helper()

	if expected.Days == nil && actual.Days == nil {
		return true
	}

	if expected.Days == nil || actual.Days == nil {
		t.Errorf("MonthState days mismatch: expected nil=%v, actual nil=%v", expected.Days == nil, actual.Days == nil)
		return false
	}

	if !assert.Equal(t, len(expected.Days), len(actual.Days), "Month should have same number of days") {
		return false
	}

	for day, expectedDayState := range expected.Days {
		actualDayState, ok := actual.Days[day]
		if !assert.True(t, ok, "Missing day %d in actual MonthState", day) {
			return false
		}
		if !assert.Equal(t, expectedDayState.State, actualDayState.State, "Day %d state mismatch", day) {
			return false
		}
	}

	return true
}

// AssertYearState compares two YearState structs
func AssertYearState(t *testing.T, expected, actual model.YearState, msgAndArgs ...interface{}) bool {
	t.Helper()

	if expected.Months == nil && actual.Months == nil {
		return true
	}

	if expected.Months == nil || actual.Months == nil {
		t.Errorf("YearState months mismatch: expected nil=%v, actual nil=%v", expected.Months == nil, actual.Months == nil)
		return false
	}

	if !assert.Equal(t, len(expected.Months), len(actual.Months), "Year should have same number of months") {
		return false
	}

	for month, expectedMonthState := range expected.Months {
		actualMonthState, ok := actual.Months[month]
		if !assert.True(t, ok, "Missing month %d in actual YearState", month) {
			return false
		}
		if !AssertMonthState(t, expectedMonthState, actualMonthState, "Month %d mismatch", month) {
			return false
		}
	}

	return true
}

// RequireYearState is like AssertYearState but fails the test immediately
func RequireYearState(t *testing.T, expected, actual model.YearState, msgAndArgs ...interface{}) {
	t.Helper()

	if expected.Months == nil && actual.Months == nil {
		return
	}

	require.NotNil(t, actual.Months, "Actual YearState.Months should not be nil")
	require.NotNil(t, expected.Months, "Expected YearState.Months should not be nil")

	require.Equal(t, len(expected.Months), len(actual.Months), "Year should have same number of months")

	for month, expectedMonthState := range expected.Months {
		actualMonthState, ok := actual.Months[month]
		require.True(t, ok, "Missing month %d in actual YearState", month)
		require.Equal(t, len(expectedMonthState.Days), len(actualMonthState.Days),
			"Month %d should have same number of days", month)

		for day, expectedDayState := range expectedMonthState.Days {
			actualDayState, ok := actualMonthState.Days[day]
			require.True(t, ok, "Missing day %d in month %d", day, month)
			require.Equal(t, expectedDayState.State, actualDayState.State,
				"Day %d of month %d state mismatch", day, month)
		}
	}
}

// AssertSchedulePreferences compares two SchedulePreferences structs
func AssertSchedulePreferences(t *testing.T, expected, actual model.SchedulePreferences, msgAndArgs ...interface{}) bool {
	t.Helper()

	success := true
	success = success && assert.Equal(t, expected.Sunday, actual.Sunday, "Sunday mismatch")
	success = success && assert.Equal(t, expected.Monday, actual.Monday, "Monday mismatch")
	success = success && assert.Equal(t, expected.Tuesday, actual.Tuesday, "Tuesday mismatch")
	success = success && assert.Equal(t, expected.Wednesday, actual.Wednesday, "Wednesday mismatch")
	success = success && assert.Equal(t, expected.Thursday, actual.Thursday, "Thursday mismatch")
	success = success && assert.Equal(t, expected.Friday, actual.Friday, "Friday mismatch")
	success = success && assert.Equal(t, expected.Saturday, actual.Saturday, "Saturday mismatch")

	return success
}

// AssertThemePreferences compares two ThemePreferences structs
func AssertThemePreferences(t *testing.T, expected, actual model.ThemePreferences, msgAndArgs ...interface{}) bool {
	t.Helper()
	return assert.Equal(t, expected, actual, msgAndArgs...)
}

// AssertNote compares two Note structs
func AssertNote(t *testing.T, expected, actual model.Note, msgAndArgs ...interface{}) bool {
	t.Helper()
	return assert.Equal(t, expected.Note, actual.Note, msgAndArgs...)
}

// AssertLinkedAccounts compares two slices of LinkedAccount
func AssertLinkedAccounts(t *testing.T, expected, actual []model.LinkedAccount, msgAndArgs ...interface{}) bool {
	t.Helper()

	if !assert.Equal(t, len(expected), len(actual), "LinkedAccounts count mismatch") {
		return false
	}

	// Create maps for easier comparison
	expectedMap := make(map[string]model.LinkedAccount)
	for _, acc := range expected {
		expectedMap[acc.Provider] = acc
	}

	actualMap := make(map[string]model.LinkedAccount)
	for _, acc := range actual {
		actualMap[acc.Provider] = acc
	}

	for provider, expectedAcc := range expectedMap {
		actualAcc, ok := actualMap[provider]
		if !assert.True(t, ok, "Missing provider %s in actual", provider) {
			return false
		}
		if !assert.Equal(t, expectedAcc, actualAcc, "LinkedAccount for provider %s mismatch", provider) {
			return false
		}
	}

	return true
}
