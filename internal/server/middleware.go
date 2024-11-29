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

func AllowedAuthMethods(authMethods ...auth.Method) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authMethod, err := getAuthMethod(r)
			if err != nil {
				err = fmt.Errorf("failed to get auth method: %w", err)
				writeError(w, internalErrorMsg, http.StatusInternalServerError)
				return
			}

			if !slices.Contains(authMethods, authMethod) {
				writeError(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (s *Server) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		method, _ := getAuthMethod(r)
		userID, _ := getUserID(r)
		slog.Info("request received", "method", r.Method, "path", r.URL.Path, "authMethod", method, "userID", userID)
		next.ServeHTTP(sw, r)
		slog.Info("request processed", "method", r.Method, "path", r.URL.Path, "status", sw.status, "duration", time.Since(start), "authMethod", method, "userID", userID)
	})
}

func injectAuth(db database.Databaser, cfger config.AppConfigurer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			val := make(ctxValue)

			switch cfg := cfger.(type) {
			case config.StandaloneApp:
				val.set(ctxAuthMethodKey, auth.MethodExcluded)
				val.set(ctxUserIDKey, 1)
			case config.IntegratedApp:
				token, authMethod := auth.GetAuth(r)
				val.set(ctxAuthMethodKey, authMethod)
				userID, err := auth.GetUserID(cfg, db, token, authMethod)
				if err != nil {
					auth.ClearCookie(cfg, w)
				}
				val.set(ctxUserIDKey, userID)
			}
			ctx = context.WithValue(ctx, ctxKey, val)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
