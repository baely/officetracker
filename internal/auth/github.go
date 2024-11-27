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

var DefaultScopes = []string{
	"state:read",
	"state:write",
	"note:read",
	"note:write",
	"developer:read",
	"developer:write",
	"report:read",
	"report:write",
}

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

		ghID, ghUser, err := getGithubData(token.AccessToken)
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

		// temp: update user with github data
		err = db.UpdateUser(userID, ghUser)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to update user: %v", err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		err = IssueToken(cfg, w, userID, DefaultScopes)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to issue token: %v", err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		slog.Info(fmt.Sprintf("logged in user: %d", userID))
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
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

func handleDemoAuth(cfg config.IntegratedApp, db database.Databaser) func(http.ResponseWriter, *http.Request) {
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
		err = IssueToken(cfg, w, userID, DefaultScopes)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to issue token: %v", err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		slog.Info(fmt.Sprintf("logged in user: %d", userID))
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}
