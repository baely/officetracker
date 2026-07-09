package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
)

func handleLogout(cfg config.IntegratedApp, author *Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		author.Logout(r.Context(), cfg, w, r)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}

func Router(cfg config.IntegratedApp, db database.Databaser, author *Auth) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/logout", handleLogout(cfg, author))
		r.Get("/callback/auth0", author.handleAuth0Callback(cfg, db))
		r.Post("/native", author.HandleNativeExchange(cfg, db))
	}
}
