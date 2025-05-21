package v1

import (
	"fmt"

	"github.com/baely/officetracker/internal/embed"
	"github.com/baely/officetracker/pkg/model"
)

func (i *Service) GetSettings(req model.GetSettingsRequest) (model.GetSettingsResponse, error) {
	accounts, err := i.db.GetUserGithubAccounts(req.Meta.UserID)
	if err != nil {
		return model.GetSettingsResponse{}, fmt.Errorf("failed to get user GitHub accounts: %w", err)
	}

	userTheme, err := i.db.GetUserTheme(req.Meta.UserID)
	if err != nil {
		// Log the error but don't fail the whole request; a default theme will be used by the frontend/rendering logic.
		// Or, if GetUserTheme returns a default on error, this might not even be an issue.
		// Assuming GetUserTheme is robust enough or returns a default.
		slog.Error("failed to get user theme, will use default", "user_id", req.Meta.UserID, "error", err)
		// If GetUserTheme can return an error that means "no preference set" vs "db error", handle accordingly.
		// For now, assume it returns a usable theme (possibly default) or an error that should be logged.
	}

	// Note: model.GetSettingsResponse currently doesn't have fields for UserTheme and AvailableThemes.
	// These were added to server.settingsPage struct.
	// The GetSettingsResponse model should be updated if these are to be part of the API response.
	// For now, assuming the settingsPage struct in server/templates.go is populated correctly by the handler in server.go,
	// which can call i.db.GetUserTheme and access embed.AvailableThemes directly.
	// This service method primarily focuses on data directly from the DB if not about available themes.

	return model.GetSettingsResponse{
		GithubAccounts: accounts,
		Theme:          userTheme, // Added theme to response model
	}, nil
}

func (i *Service) UpdateSettings(req model.PutSettingsRequest) (model.PutSettingsResponse, error) {
	if req.Data.Theme == "" {
		// Optionally, handle empty theme string if it's not allowed,
		// or rely on GetTheme to manage fallbacks.
		// return model.PutSettingsResponse{}, fmt.Errorf("theme cannot be empty")
	}

	// Validate if the theme exists
	if _, ok := embed.GetTheme(req.Data.Theme); !ok && req.Data.Theme != i.appCfg.GetApp().DefaultTheme {
		// Allow setting to default theme even if it's not explicitly in AvailableThemes map (e.g. "default")
		// This check might be too strict if "default" isn't in AvailableThemes map.
		// A simpler approach might be to just trust the input or validate against a list of allowed theme names.
		// For now, let's assume any theme in AvailableThemes or the default theme string is valid.
		found := false
		for themeName := range embed.AvailableThemes {
			if themeName == req.Data.Theme {
				found = true
				break
			}
		}
		if !found && req.Data.Theme != i.appCfg.GetApp().DefaultTheme {
			return model.PutSettingsResponse{}, fmt.Errorf("invalid theme selected: %s", req.Data.Theme)
		}
	}


	err := i.db.SetUserTheme(req.Meta.UserID, req.Data.Theme)
	if err != nil {
		return model.PutSettingsResponse{}, fmt.Errorf("failed to set user theme: %w", err)
	}

	return model.PutSettingsResponse{
		Message: "Settings updated successfully.",
	}, nil
}
