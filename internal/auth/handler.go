package auth

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(userCookie)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to get cookie: %v", err))
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		if err = cookie.Valid(); err != nil {
			slog.Error(fmt.Sprintf("failed to validate cookie: %v", err))
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		err = validateToken(cookie.Value)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to validate token: %v", err))
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func Router() func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/callback/github", handleGithubCallback)
	}
}
