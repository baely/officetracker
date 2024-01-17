package auth

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/baely/officetracker/internal/util"
)

const (
	githubUserEndpoint = "https://api.github.com/user"
)

type GithubUserResponse struct {
	Login string `json:"login"`
	Id    int    `json:"id"`
}

func ghOauthCfg() *oauth2.Config {
	ghClientID := os.Getenv("GH_CLIENT_ID")
	ghSecret := os.Getenv("GH_SECRET")
	return &oauth2.Config{
		ClientID:     ghClientID,
		ClientSecret: ghSecret,
		Endpoint:     github.Endpoint,
		RedirectURL:  fmt.Sprintf("%sauth/callback/github", util.BaseUri()),
		Scopes:       []string{"read:user"},
	}
}

func GitHubAuthUri() string {
	state := "state"
	slog.Info(fmt.Sprintf("redirecting to github with state: %s", state))
	slog.Info(fmt.Sprintf("redirecting to github: %+v", ghOauthCfg()))
	return ghOauthCfg().AuthCodeURL(state)
}

func handleGithubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		slog.Error("no code provided")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	token, err := ghOauthCfg().Exchange(r.Context(), code)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to exchange code: %v", err))
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	userID, err := getGithubData(token.AccessToken)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to get github data: %v", err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = issueToken(w, userID)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to issue token: %v", err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/setup", http.StatusTemporaryRedirect)
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
