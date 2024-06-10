package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
)

func AllowedAuthMethods(authMethods ...AuthMethod) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authMethod, err := getAuthMethod(r)
			if err != nil {
				err = fmt.Errorf("failed to get auth method: %w", err)
				//http.Error(w, internalErrorMsg, http.StatusInternalServerError)
				writeError(w, internalErrorMsg, http.StatusInternalServerError)
				return
			}

			if !slices.Contains(authMethods, authMethod) {
				//http.Error(w, "unauthorized", http.StatusUnauthorized)
				writeError(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info(fmt.Sprintf("request: %s %s took %s", r.Method, r.URL.Path, time.Since(start)))
	})
}

func injectAuth(db database.Databaser, cfgIface config.AppConfigurer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			val := make(ctxValue)

			switch cfg := cfgIface.(type) {
			case config.StandaloneApp:
				val.set(ctxAuthMethodKey, AuthMethodExcluded)
				val.set(ctxUserIDKey, "42069")
			case config.IntegratedApp:
				userID := auth.GetUserID(db, cfg, w, r)
				if userID != 0 {
					val.set(ctxAuthMethodKey, AuthMethodSSO)
					val.set(ctxUserIDKey, userID)
				} else {
					userID = auth.GetUserFromSecret(db, r)
					if userID != 0 {
						val.set(ctxAuthMethodKey, AuthMethodSecret)
						val.set(ctxUserIDKey, userID)
					} else {
						val.set(ctxAuthMethodKey, AuthMethodNone)
					}
				}
			}
			ctx = context.WithValue(ctx, ctxKey, val)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
