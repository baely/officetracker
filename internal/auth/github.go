package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/util"
)

type GithubUserResponse struct {
	Login string `json:"login"`
	Id    int    `json:"id"`
}

func ClearCookie(cfg config.IntegratedApp, w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName(cfg),
		Value:    "",
		Path:     util.BasePath(cfg.Domain),
		Expires:  time.Unix(0, 0),
		Domain:   util.QualifiedDomain(cfg.Domain),
		HttpOnly: true,
		Secure:   false,
	})
}

func toUserID(db database.Databaser, ghID string) (int, error) {
	userID, err := db.GetUserByGHID(ghID)
	if errors.Is(err, database.ErrNoUser) {
		userID, err = db.SaveUserByGHID(ghID)
	}
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func subjectToUserID(db database.Databaser, profile Auth0Profile) (int, error) {
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal profile: %w", err)
	}
	profileStr := string(profileJSON)

	// Tier 1: Check auth0_users table
	userID, err := db.GetUserByAuth0Sub(profile.Sub)
	if err == nil {
		db.UpdateAuth0Profile(profile.Sub, profileStr)
		return userID, nil
	}
	if !errors.Is(err, database.ErrNoUser) {
		return 0, err
	}

	// Tier 2: Check gh_users table (migration fallback)
	provider, providerID, err := parseAuth0Subject(profile.Sub)
	if err != nil {
		return 0, err
	}

	if provider == "github" {
		existingUserID, err := db.GetUserByGHID(providerID)
		if err == nil {
			// Found existing GitHub user - migrate by linking Auth0 identity
			err = db.LinkAuth0Account(existingUserID, profile.Sub, profileStr)
			if err != nil {
				return 0, fmt.Errorf("failed to migrate github user to auth0: %w", err)
			}
			return existingUserID, nil
		}
		if !errors.Is(err, database.ErrNoUser) {
			return 0, err
		}
	}

	// Tier 3: New user signup
	return db.SaveUserByAuth0Sub(profile.Sub, profileStr)
}

func parseAuth0Subject(sub string) (provider string, identifier string, err error) {
	parts := strings.Split(sub, "|")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid subject format")
	}
	return parts[0], parts[1], nil
}

func handleDemoAuth(cfg config.IntegratedApp, db database.Databaser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !cfg.App.Demo {
			slog.Error("demo auth called on non-demo app")
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		userID, err := toUserID(db, demoUserId)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to get user id: %v", err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		err = issueToken(cfg, w, userID)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to issue token: %v", err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		slog.Info(fmt.Sprintf("logged in user: %d", userID))
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}
