package auth

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
)

func handleLogout(cfg config.IntegratedApp) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ClearCookie(cfg, w)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}

func Router(cfg config.IntegratedApp, db database.Databaser, redis *database.Redis, author *Auth) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/logout", handleLogout(cfg))
		r.Get("/callback/auth0", author.handleAuth0Callback(cfg, db))
		r.Get("/demo", handleDemoAuth(cfg, db))
	}
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
