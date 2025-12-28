package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/baely/officetracker/pkg/model"
	"github.com/baely/officetracker/testutil/mocks"
)

func TestGetSettings(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	// Setup test data
	themePrefs := model.ThemePreferences{
		Theme:            "dark",
		WeatherEnabled:   true,
		TimeBasedEnabled: false,
		Location:         "Sydney",
	}
	schedulePrefs := model.SchedulePreferences{
		Monday:    model.StateWorkFromHome,
		Tuesday:   model.StateWorkFromOffice,
		Wednesday: model.StateWorkFromHome,
		Thursday:  model.StateWorkFromOffice,
		Friday:    model.StateWorkFromHome,
	}
	mockDB.SaveThemePreferences(1, themePrefs)
	mockDB.SaveSchedulePreferences(1, schedulePrefs)
	mockDB.AddUser(1)
	mockDB.SetUserSuspended(1, false)
	req := model.GetSettingsRequest{
		Meta: model.GetSettingsRequestMeta{
			UserID: 1,
		},
	}

	resp, err := service.GetSettings(req)
	require.NoError(t, err)

	assert.Equal(t, themePrefs, resp.ThemePreferences)
	assert.Equal(t, schedulePrefs, resp.SchedulePreferences)
	// LinkedAccounts would be tested when we have the DB method working properly
}

func TestUpdateThemePreferences(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	themePrefs := model.ThemePreferences{
		Theme:            "light",
		WeatherEnabled:   false,
		TimeBasedEnabled: true,
		Location:         "Melbourne",
	}

	req := model.UpdateThemePreferencesRequest{
		Meta: model.UpdateThemePreferencesRequestMeta{
			UserID: 1,
		},
		Data: themePrefs,
	}

	_, err := service.UpdateThemePreferences(req)
	require.NoError(t, err)

	// Verify preferences were saved
	assert.Len(t, mockDB.SaveThemePreferencesCalls, 1)
	assert.Equal(t, 1, mockDB.SaveThemePreferencesCalls[0].UserID)
	assert.Equal(t, themePrefs, mockDB.SaveThemePreferencesCalls[0].Prefs)
}

func TestUpdateSchedulePreferences(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	schedulePrefs := model.SchedulePreferences{
		Monday:    model.StateWorkFromOffice,
		Tuesday:   model.StateWorkFromOffice,
		Wednesday: model.StateWorkFromHome,
		Thursday:  model.StateWorkFromOffice,
		Friday:    model.StateWorkFromHome,
		Saturday:  model.StateUntracked,
		Sunday:    model.StateUntracked,
	}

	req := model.UpdateSchedulePreferencesRequest{
		Meta: model.UpdateSchedulePreferencesRequestMeta{
			UserID: 1,
		},
		Data: schedulePrefs,
	}

	_, err := service.UpdateSchedulePreferences(req)
	require.NoError(t, err)

	// Verify preferences were saved
	assert.Len(t, mockDB.SaveSchedulePreferencesCalls, 1)
	assert.Equal(t, 1, mockDB.SaveSchedulePreferencesCalls[0].UserID)
	assert.Equal(t, schedulePrefs, mockDB.SaveSchedulePreferencesCalls[0].Prefs)
}

func TestUpdateThemePreferences_ErrorHandling(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnSaveThemePreferences = assert.AnError
	service := New(mockDB, nil)

	req := model.UpdateThemePreferencesRequest{
		Meta: model.UpdateThemePreferencesRequestMeta{
			UserID: 1,
		},
		Data: model.ThemePreferences{Theme: "dark"},
	}

	_, err := service.UpdateThemePreferences(req)
	assert.Error(t, err)
}

func TestUpdateSchedulePreferences_ErrorHandling(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnSaveSchedulePreferences = assert.AnError
	service := New(mockDB, nil)

	req := model.UpdateSchedulePreferencesRequest{
		Meta: model.UpdateSchedulePreferencesRequestMeta{
			UserID: 1,
		},
		Data: model.SchedulePreferences{Monday: model.StateWorkFromHome},
	}

	_, err := service.UpdateSchedulePreferences(req)
	assert.Error(t, err)
}
