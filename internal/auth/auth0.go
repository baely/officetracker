package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
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
	// Generate a secure random state using crypto/rand
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", fmt.Errorf("failed to generate state: %v", err)
	}
	state := base64.URLEncoding.EncodeToString(stateBytes)

	// Store the state in Redis with 0 as userID (new user), expiring in 10 minutes
	key := fmt.Sprintf("auth0:state:%s", state)
	err := a.redis.SetState(context.Background(), key, 0, 10*time.Minute)
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

		var profile map[string]interface{}
		if err := idToken.Claims(&profile); err != nil {
			slog.Error(fmt.Sprintf("failed to parse claims: %v", err))
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		// print the user id for now and claims
		slog.Info("Auth0 login successful", "userID", existingUserID, "claims", profile)

		var userID int
		if existingUserID != 0 {
			slog.Error("non 0 userId retrived from redis for auth0 login")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		} else {
			subject, ok := profile["sub"]
			if !ok {
				slog.Error("failed to retrieve subject")
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			subjectString, ok := subject.(string)
			if !ok {
				slog.Error("subject not in string format. format: %T", subject)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			userID, err = subjectToUserID(db, subjectString)
			if err != nil {
				slog.Error(fmt.Sprintf("failed to get/create user: %v", err))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
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
