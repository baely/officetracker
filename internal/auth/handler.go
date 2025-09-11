package auth

import (
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
		r.Get("/generate-gh", handleGenerateGithub(cfg, redis))
		r.Get("/callback/auth0", author.handleAuth0Callback())
		r.Get("/callback/github", handleGithubCallback(cfg, db, redis))
		r.Get("/demo", handleDemoAuth(cfg, db))
	}
}
