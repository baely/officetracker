package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/util"
)

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

func ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    userCookie,
		Value:   "",
		Expires: time.Unix(0, 0),
	})
}

func handleGithubCallback(cfg config.IntegratedApp, db database.Databaser) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			slog.Error("no code provided")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		token, err := ghOauthCfg(cfg).Exchange(r.Context(), code)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to exchange code: %v", err))
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		ghID, err := getGithubData(token.AccessToken)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to get github data: %v", err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		userID, err := toUserID(db, ghID)
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

		http.Redirect(w, r, "/setup", http.StatusTemporaryRedirect)
	}
}

func getGithubData(accessToken string) (string, error) {
	req, err := http.NewRequest("GET", githubUserEndpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	var user GithubUserResponse
	if err = json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", user.Id), nil
}

func toUserID(db database.Databaser, ghID string) (string, error) {
	userID, err := db.GetUserByGHID(ghID)
	if errors.Is(err, database.ErrNoUser) {
		userID, err = db.SaveUser(ghID)
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%05d", userID), nil
}

func handleFake(cfg config.IntegratedApp, db database.Databaser) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !cfg.App.Demo {
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
		http.Redirect(w, r, "/setup", http.StatusTemporaryRedirect)
	}
}
