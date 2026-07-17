package v1

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/baely/officetracker/internal/util"
	"github.com/baely/officetracker/pkg/model"
)

// providerDisplayNames maps Auth0 provider identifiers to human-readable names
var providerDisplayNames = map[string]string{
	"github":         "GitHub",
	"google-oauth2":  "Google",
	"auth0":          "Email",
	"microsoft":      "Microsoft",
	"apple":          "Apple",
	"twitter":        "Twitter",
	"facebook":       "Facebook",
	"linkedin":       "LinkedIn",
}

func (i *Service) GetSettings(req model.GetSettingsRequest) (model.GetSettingsResponse, error) {
	linkedAccounts, err := i.db.GetUserLinkedAccounts(req.Meta.UserID)
	if err != nil {
		return model.GetSettingsResponse{}, err
	}

	// Add display names to linked accounts
	caser := cases.Title(language.English)
	for i, account := range linkedAccounts {
		if displayName, ok := providerDisplayNames[account.Provider]; ok {
			linkedAccounts[i].ProviderDisplay = displayName
		} else {
			// Fallback: title case the provider name
			linkedAccounts[i].ProviderDisplay = caser.String(account.Provider)
		}
	}

	themePrefs, err := i.db.GetThemePreferences(req.Meta.UserID)
	if err != nil {
		return model.GetSettingsResponse{}, err
	}

	schedulePrefs, err := i.db.GetSchedulePreferences(req.Meta.UserID)
	if err != nil {
		return model.GetSettingsResponse{}, err
	}

	calendarPrefs, err := i.db.GetCalendarPreferences(req.Meta.UserID)
	if err != nil {
		return model.GetSettingsResponse{}, err
	}
	calendarPrefs.TrackingYearStartMonth = util.NormaliseStartMonth(calendarPrefs.TrackingYearStartMonth)

	targetPrefs, err := i.db.GetTargetPreferences(req.Meta.UserID)
	if err != nil {
		return model.GetSettingsResponse{}, err
	}

	return model.GetSettingsResponse{
		LinkedAccounts:      linkedAccounts,
		ThemePreferences:    themePrefs,
		SchedulePreferences: schedulePrefs,
		CalendarPreferences: calendarPrefs,
		TargetPreferences:   targetPrefs,
	}, nil
}

func (i *Service) UpdateThemePreferences(req model.UpdateThemePreferencesRequest) (model.UpdateThemePreferencesResponse, error) {
	err := i.db.SaveThemePreferences(req.Meta.UserID, req.Data)
	return model.UpdateThemePreferencesResponse{}, err
}

func (i *Service) UpdateSchedulePreferences(req model.UpdateSchedulePreferencesRequest) (model.UpdateSchedulePreferencesResponse, error) {
	err := i.db.SaveSchedulePreferences(req.Meta.UserID, req.Data)
	return model.UpdateSchedulePreferencesResponse{}, err
}

func (i *Service) UpdateCalendarPreferences(req model.UpdateCalendarPreferencesRequest) (model.UpdateCalendarPreferencesResponse, error) {
	req.Data.TrackingYearStartMonth = util.NormaliseStartMonth(req.Data.TrackingYearStartMonth)
	err := i.db.SaveCalendarPreferences(req.Meta.UserID, req.Data)
	return model.UpdateCalendarPreferencesResponse{}, err
}

func (i *Service) UpdateTargetPreferences(req model.UpdateTargetPreferencesRequest) (model.UpdateTargetPreferencesResponse, error) {
	req.Data.DefaultTargetPercent = util.ClampTargetPercent(req.Data.DefaultTargetPercent)
	err := i.db.SaveTargetPreferences(req.Meta.UserID, req.Data)
	return model.UpdateTargetPreferencesResponse{}, err
}
