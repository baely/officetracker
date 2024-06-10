package auth

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
)

func handleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ClearCookie(w)
		slog.Info("logged out")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}

func Router(cfg config.IntegratedApp, db database.Databaser) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/logout", handleLogout())
		r.Get("/callback/github", handleGithubCallback(cfg, db))
		r.Get("/demo", handleDemoAuth(cfg, db))
	}
}
