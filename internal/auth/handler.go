package auth

import (
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

// handleLogin starts the Auth0 SSO flow. The SSO URL embeds a per-request state
// nonce (stored in Redis), so it cannot be baked into the static, cacheable
// login page; the page links here instead and we redirect to Auth0 on demand.
func handleLogin(author *Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uri, err := author.Auth0SSOUri()
		if err != nil {
			slog.Error("failed to generate SSO URI: " + err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Cache-Control", "private, no-store")
		http.Redirect(w, r, uri, http.StatusTemporaryRedirect)
	}
}

func Router(cfg config.IntegratedApp, db database.Databaser, author *Auth) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/login", handleLogin(author))
		r.Get("/logout", handleLogout(cfg))
		r.Get("/callback/auth0", author.handleAuth0Callback(cfg, db))
		r.Post("/native", author.HandleNativeExchange(cfg, db))
	}
}
