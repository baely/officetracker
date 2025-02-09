package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/context"
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/util"
)

func getUserID(r *http.Request) (int, error) {
	userID, ok := context.GetCtxValue(r).Get(context.CtxUserIDKey).(int)
	if !ok {
		return 0, fmt.Errorf("no user id in context")
	}
	return userID, nil
}

const (
	githubUserEndpoint = "https://api.github.com/user"
)

type GithubUserResponse struct {
	Login string `json:"login"`
	Id    int    `json:"id"`
}

func ghOauthCfg(cfg config.IntegratedApp) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.Github.ClientID,
		ClientSecret: cfg.Github.Secret,
		Endpoint:     github.Endpoint,
		RedirectURL:  fmt.Sprintf("%sauth/callback/github", util.BaseUri(cfg)),
		Scopes:       []string{"read:user"},
	}
}

func SSOUri(cfg config.IntegratedApp) string {
	if cfg.App.Demo {
		return "/auth/demo"
	}

	state := "state"
	return ghOauthCfg(cfg).AuthCodeURL(state)
}

func ClearCookie(cfg config.IntegratedApp, w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     userCookie,
		Value:    "",
		Path:     util.BasePath(cfg.Domain),
		Expires:  time.Unix(0, 0),
		Domain:   util.QualifiedDomain(cfg.Domain),
		HttpOnly: true,
		Secure:   false,
	})
}

func handleGenerateGithub(cfg config.IntegratedApp, redis *database.Redis) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := getUserID(r)
		if err != nil || userID == 0 {
			slog.Error("no user id in context")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Generate a secure random state
		stateBytes := make([]byte, 32)
		if _, err := rand.Read(stateBytes); err != nil {
			slog.Error(fmt.Sprintf("failed to generate state: %v", err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		state := base64.URLEncoding.EncodeToString(stateBytes)

		// Store the state in Redis with the user ID, expiring in 10 minutes
		key := fmt.Sprintf("github:state:%s", state)
		err = redis.SetState(r.Context(), key, userID, 10*time.Minute)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to store state: %v", err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Generate the GitHub OAuth URL with the state
		authURL := ghOauthCfg(cfg).AuthCodeURL(state)

		// Return the URL to the client
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"url": authURL,
		})
	}
}

func handleGithubCallback(cfg config.IntegratedApp, db database.Databaser, redis *database.Redis) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			slog.Error("no code provided")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		state := r.URL.Query().Get("state")
		var existingUserID int
		var err error

		// If state exists, validate it's a linking flow
		if state != "state" {
			key := fmt.Sprintf("github:state:%s", state)
			existingUserID, err = redis.GetStateInt(r.Context(), key)
			if err != nil {
				slog.Error(fmt.Sprintf("invalid or expired state: %v", err))
				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
				return
			}
			// Delete the state key since it's been used
			_ = redis.DeleteState(r.Context(), key)
		}

		token, err := ghOauthCfg(cfg).Exchange(r.Context(), code)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to exchange code: %v", err))
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		ghID, ghUser, err := getGithubData(token.AccessToken)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to get github data: %v", err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		var userID int
		if existingUserID != 0 {
			// Account linking flow - update existing user's GitHub info
			err = db.UpdateUserGithub(existingUserID, ghID, ghUser)
			if err != nil {
				if err.Error() == "github account already associated with another user" {
					slog.Error(fmt.Sprintf("github account already linked: %v", err))
					http.Error(w, "This GitHub account is already linked to another Officetracker account", http.StatusConflict)
					return
				}
				slog.Error(fmt.Sprintf("failed to update user github: %v", err))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			userID = existingUserID
			slog.Info(fmt.Sprintf("linked github account for user: %d", userID))
		} else {
			// Fresh login flow - create or get user by GitHub ID
			userID, err = toUserID(db, ghID)
			if err != nil {
				slog.Error(fmt.Sprintf("failed to get/create user: %v", err))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			// Update username in case it changed
			err = db.UpdateUser(userID, ghUser)
			if err != nil {
				slog.Error(fmt.Sprintf("failed to update username: %v", err))
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

		// Redirect to settings if it was a linking flow, otherwise to home
		if existingUserID != 0 {
			http.Redirect(w, r, "/settings", http.StatusTemporaryRedirect)
		} else {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		}
	}
}

func getGithubData(accessToken string) (userID string, username string, err error) {
	req, err := http.NewRequest("GET", githubUserEndpoint, nil)
	if err != nil {
		return
	}
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	var user GithubUserResponse
	if err = json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return
	}

	userID = fmt.Sprintf("%d", user.Id)
	username = user.Login
	return
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
