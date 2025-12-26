package v1

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

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

	return model.GetSettingsResponse{
		LinkedAccounts:      linkedAccounts,
		ThemePreferences:    themePrefs,
		SchedulePreferences: schedulePrefs,
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
