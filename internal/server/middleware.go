package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/config"
	context2 "github.com/baely/officetracker/internal/context"
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
			val := make(context2.CtxValue)

			switch cfg := cfger.(type) {
			case config.StandaloneApp:
				val.Set(context2.CtxAuthMethodKey, auth.MethodExcluded)
				val.Set(context2.CtxUserIDKey, 1)
			case config.IntegratedApp:
				token, authMethod := auth.GetAuth(r)
				val.Set(context2.CtxAuthMethodKey, authMethod)
				userID, err := auth.GetUserID(cfg, db, token, authMethod)
				if err != nil {
					auth.ClearCookie(cfg, w)
				}
				val.Set(context2.CtxUserIDKey, userID)
			}
			ctx = context.WithValue(ctx, context2.CtxKey, val)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func Otel(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := findMatchedRoute(r)
		h := otelhttp.NewMiddleware(route)
		h(next).ServeHTTP(w, r)
	})
}

func findMatchedRoute(r *http.Request) string {
	var matchedPattern string
	router := r.Context().Value(chi.RouteCtxKey).(*chi.Context).Routes

	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if method == r.Method {
			// Create regex pattern from the route
			rp := regexp.MustCompile("\\{[^\\}]*\\}")
			routeRegex := "^" + rp.ReplaceAllString(route, "([^/]+)") + "$"

			if match, _ := regexp.MatchString(routeRegex, r.URL.Path); match {
				matchedPattern = route
			}
		}
		return nil
	}

	_ = chi.Walk(router, walkFunc)
	return matchedPattern
}

func checkSuspension(db database.Databaser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, err := getUserID(r)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			suspended, err := db.IsUserSuspended(userID)
			if err != nil {
				slog.Error("failed to check user suspension status", "userID", userID, "error", err)
				writeError(w, internalErrorMsg, http.StatusInternalServerError)
				return
			}

			if suspended {
				http.Redirect(w, r, "/suspended", http.StatusSeeOther)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
