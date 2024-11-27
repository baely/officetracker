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

func RequiredScopes(requiredScopes ...Scope) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scopes, err := getScopes(r)
			if err != nil {
				err = fmt.Errorf("failed to get scopes: %w", err)
				slog.Error(err.Error())
				writeError(w, internalErrorMsg, http.StatusInternalServerError)
				return
			}

			if !compareScopes(requiredScopes, scopes) {
				slog.Error("unauthorized")
				writeError(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			handler.ServeHTTP(w, r)
		})
	}
}

type StatusWriter struct {
	http.ResponseWriter
	status int
}

func (w *StatusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (s *Server) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &StatusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		if sw.status == 0 {
			sw.status = http.StatusOK
		}
		slog.Info(fmt.Sprintf("request: %d %s %s took %s", sw.status, r.Method, r.URL.Path, time.Since(start)))
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
				val.set(ctxUserIDKey, 1)
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
						val.set(ctxAuthMethodKey, AuthMethodAnonymous)
					}
				}
			}
			ctx = context.WithValue(ctx, ctxKey, val)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func copyScopes(cfgIface config.AppConfigurer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			val := getCtxValue(r)

			scopes, _ := auth.GetScopes(r)

			// temp: if no scopes, re-issue token
			if len(scopes) == 0 {
				userID, _ := getUserID(r)
				_ = auth.IssueToken(config.IntegratedApp{}, w, userID, auth.DefaultScopes)
				scopes = auth.DefaultScopes
			}

			val.set(ctxScopesKey, scopes)

			ctx = context.WithValue(ctx, ctxKey, val)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
