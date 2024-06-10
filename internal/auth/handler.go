package auth

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
)

func Middleware(cfg config.IntegratedApp, db database.Databaser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(userCookie)
			if err != nil {
				slog.Error(fmt.Sprintf("failed to get cookie: %v", err))
				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
				return
			}
			if err = cookie.Valid(); err != nil {
				slog.Error(fmt.Sprintf("invalid cookie %v", err))
				http.Redirect(w, r, "/logout", http.StatusTemporaryRedirect)
				return
			}

			err = validateToken(cfg, cookie.Value)
			if err != nil {
				slog.Error(fmt.Sprintf("failed to validate token: %v", err))
				http.Redirect(w, r, "/logout", http.StatusTemporaryRedirect)
				return
			}

			_, err = validUser(db, cfg, cookie.Value)
			if err != nil {
				slog.Error(fmt.Sprintf("failed to validate user: %v", err))
				http.Redirect(w, r, "/logout", http.StatusTemporaryRedirect)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

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
