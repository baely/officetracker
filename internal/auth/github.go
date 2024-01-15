package auth

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var (
	ghClientID = os.Getenv("GH_CLIENT_ID")
	ghSecret   = os.Getenv("GH_SECRET")
	ghOauthCfg = &oauth2.Config{
		ClientID:     ghClientID,
		ClientSecret: ghSecret,
		Endpoint:     github.Endpoint,
		RedirectURL:  "http://localhost:8080/auth/callback/github",
		Scopes:       []string{"read:user"},
	}
)

type GithubUserResponse struct {
	Login string `json:"login"`
	Id    int    `json:"id"`
}

func githubRedirect(w http.ResponseWriter, r *http.Request) {
	state := "state"
	url := ghOauthCfg.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleGithubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		slog.Error("no code provided")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	token, err := ghOauthCfg.Exchange(r.Context(), code)
	if err != nil {
		githubRedirect(w, r)
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

	http.Redirect(w, r, "http://localhost:8080/", http.StatusTemporaryRedirect)
}

func getGithubData(accessToken string) (string, error) {
	endpoint := "https://api.github.com/user"
	req, err := http.NewRequest("GET", endpoint, nil)
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
