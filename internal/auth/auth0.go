package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
)

type Auth0Profile struct {
	Sub string `json:"sub"`

	Nickname string `json:"nickname,omitempty"` // Username displayed in UI
	Picture  string `json:"picture,omitempty"`  // Avatar URL displayed in UI
}

func (a *Auth) Auth0OauthCfg() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     a.auth0Cfg.ClientID,
		ClientSecret: a.auth0Cfg.ClientSecret,
		Endpoint:     a.provider.Endpoint(),
		RedirectURL:  fmt.Sprintf("%sauth/callback/auth0", a.baseUri),
		Scopes:       []string{oidc.ScopeOpenID, "profile"},
	}
}

func (a *Auth) Auth0SSOUri() (string, error) {
	return a.GenerateAuth0AuthLink(0)
}

func (a *Auth) GenerateAuth0AuthLink(userId int) (string, error) {
	// Generate a secure random state using crypto/rand
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", fmt.Errorf("failed to generate state: %v", err)
	}
	state := base64.URLEncoding.EncodeToString(stateBytes)

	// Store the state in Redis with 0 as userID (new user), expiring in 10 minutes
	key := fmt.Sprintf("auth0:state:%s", state)
	err := a.redis.SetState(context.Background(), key, userId, 10*time.Minute)
	if err != nil {
		return "", fmt.Errorf("failed to store state: %v", err)
	}

	return a.Auth0OauthCfg().AuthCodeURL(state,
		oauth2.SetAuthURLParam("prompt", "login"),
	), nil
}

func (a *Auth) handleAuth0Callback(cfg config.IntegratedApp, db database.Databaser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		code := r.URL.Query().Get("code")
		if code == "" {
			slog.Error("no code provided")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		state := r.URL.Query().Get("state")
		if state == "" {
			slog.Error("no state provided")
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		// Validate state for all flows
		key := fmt.Sprintf("auth0:state:%s", state)
		existingUserID, err := a.redis.GetStateInt(ctx, key)
		if err != nil {
			slog.Error(fmt.Sprintf("invalid or expired state: %v", err))
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		// Delete the state key since it's been used
		_ = a.redis.DeleteState(ctx, key)

		token, err := a.Auth0OauthCfg().Exchange(ctx, code)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to exchange code: %v", err))
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		idToken, err := a.verifyIDToken(ctx, token)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to verify ID token: %v", err))
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		var profile Auth0Profile
		if err := idToken.Claims(&profile); err != nil {
			slog.Error(fmt.Sprintf("failed to parse claims: %v", err))
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		if profile.Sub == "" {
			slog.Error("failed to retrieve subject from claims")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		var userID int
		if existingUserID != 0 {
			// Account linking flow - update existing user's social info
			err = a.addLoginToUser(existingUserID, profile)
			if err != nil {
				if err.Error() == "auth0 account already associated with another user" {
					slog.Error(fmt.Sprintf("auth0 account already linked: %v", err))
					http.Error(w, "This account is already linked to another Officetracker account", http.StatusConflict)
					return
				}
				slog.Error(fmt.Sprintf("failed to update user social: %v", err))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			userID = existingUserID
			slog.Info(fmt.Sprintf("linked social account for user: %d", userID))
		} else {
			userID, err = subjectToUserID(db, profile)
			if err != nil {
				slog.Error(fmt.Sprintf("failed to get/create user: %v", err))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Update profile in case it changed
			err = a.updateLoginForUser(userID, profile)
			if err != nil {
				slog.Error(fmt.Sprintf("failed to update profile: %v", err))
				// Non-critical error, continue
			}

			slog.Info(fmt.Sprintf("logged in user: %d", userID))
		}

		err = issueToken(cfg, w, userID)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to issue token: %v", err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Redirect to home page for now
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// verifyIDToken verifies that an *oauth2.Token is a valid *oidc.IDToken.
func (a *Auth) verifyIDToken(ctx context.Context, token *oauth2.Token) (*oidc.IDToken, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("no id_token field in oauth2 token")
	}

	oidcConfig := &oidc.Config{
		ClientID: a.auth0Cfg.ClientID,
	}

	return a.provider.Verifier(oidcConfig).Verify(ctx, rawIDToken)
}

func (a *Auth) addLoginToUser(existingUserID int, profile Auth0Profile) error {
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}
	return a.db.LinkAuth0Account(existingUserID, profile.Sub, string(profileJSON))
}

func (a *Auth) updateLoginForUser(userID int, profile Auth0Profile) error {
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}
	return a.db.UpdateAuth0Profile(profile.Sub, string(profileJSON))
}
