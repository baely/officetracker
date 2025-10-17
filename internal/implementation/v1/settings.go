package v1

import "github.com/baely/officetracker/pkg/model"

// GetSettings godoc
//
//	@Summary		Get user settings
//	@Description	Retrieve all user settings including GitHub accounts, theme preferences, and schedule preferences
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	model.GetSettingsResponse
//	@Failure		400	{object}	model.Error
//	@Failure		500	{object}	model.Error
//	@Security		BearerAuth
//	@Security		CookieAuth
//	@Router			/settings/ [get]
func (i *Service) GetSettings(req model.GetSettingsRequest) (model.GetSettingsResponse, error) {
	accounts, err := i.db.GetUserGithubAccounts(req.Meta.UserID)
	if err != nil {
		return model.GetSettingsResponse{}, err
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
		GithubAccounts:      accounts,
		ThemePreferences:    themePrefs,
		SchedulePreferences: schedulePrefs,
	}, nil
}

// UpdateThemePreferences godoc
//
//	@Summary		Update theme preferences
//	@Description	Update user theme preferences including theme selection and weather/time-based settings
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Param			data	body		model.ThemePreferences	true	"Theme preferences"
//	@Success		200		{object}	model.UpdateThemePreferencesResponse
//	@Failure		400		{object}	model.Error
//	@Failure		500		{object}	model.Error
//	@Security		BearerAuth
//	@Security		CookieAuth
//	@Router			/settings/theme [put]
func (i *Service) UpdateThemePreferences(req model.UpdateThemePreferencesRequest) (model.UpdateThemePreferencesResponse, error) {
	err := i.db.SaveThemePreferences(req.Meta.UserID, req.Data)
	return model.UpdateThemePreferencesResponse{}, err
}

// UpdateSchedulePreferences godoc
//
//	@Summary		Update schedule preferences
//	@Description	Update user weekly schedule preferences for default attendance states
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Param			data	body		model.SchedulePreferences	true	"Schedule preferences"
//	@Success		200		{object}	model.UpdateSchedulePreferencesResponse
//	@Failure		400		{object}	model.Error
//	@Failure		500		{object}	model.Error
//	@Security		BearerAuth
//	@Security		CookieAuth
//	@Router			/settings/schedule [put]
func (i *Service) UpdateSchedulePreferences(req model.UpdateSchedulePreferencesRequest) (model.UpdateSchedulePreferencesResponse, error) {
	err := i.db.SaveSchedulePreferences(req.Meta.UserID, req.Data)
	return model.UpdateSchedulePreferencesResponse{}, err
}
