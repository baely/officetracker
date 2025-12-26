package auth

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
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

func subjectToUserID(db database.Databaser, sub string) (int, error) {
	userId, err := validateAuth0Subject(sub)
	if err != nil {
		return 0, fmt.Errorf("invalid subject: %v", err)
	}

	return toUserID(db, userId)
}

// TODO: gracefully handle arbitrary social login providers
var validProviders = []string{
	"github",
}

func validateAuth0Subject(sub string) (string, error) {
	parts := strings.Split(sub, "|")

	if len(parts) != 2 {
		return "", fmt.Errorf("invalid number of sub parts")
	}

	provider := parts[0]
	identifier := parts[1]

	if !slices.Contains(validProviders, provider) {
		return "", fmt.Errorf("invalid social login: %s", provider)
	}

	return identifier, nil
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
