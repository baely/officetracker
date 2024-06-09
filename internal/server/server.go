package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
)

const (
	internalErrorMsg = "Internal server error"
)

type ctxValue map[string]interface{}

const (
	ctxKey           = "ctx"
	ctxUserIDKey     = "userID"
	ctxAuthMethodKey = "auth"
)

type AuthMethod int

const (
	AuthMethodUnknown = AuthMethod(iota)
	AuthMethodNone
	AuthMethodSSO
	AuthMethodSecret
	AuthMethodExcluded
)

type Server struct {
	http.Server
	cfg config.AppConfigurer
	db  database.Databaser
}

func (s *Server) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(fmt.Sprintf("request: %s %s", r.Method, r.URL.Path))
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info(fmt.Sprintf("request: %s %s took %s", r.Method, r.URL.Path, time.Since(start)))
	})
}

func (s *Server) getUserID(r *http.Request) int {
	switch cfg := s.cfg.(type) {
	case config.IntegratedApp:
		return auth.GetUserID(cfg, r)
	case config.StandaloneApp:
		return 42069
	default:
		return 0
	}
}

func NewServer(cfg config.IntegratedApp, db database.Databaser) (*Server, error) {
	s := &Server{
		db:  db,
		cfg: cfg,
	}

	r := chi.NewMux().With(s.logRequest, injectAuth(db, cfg))

	// API routes
	r.Route("/api/v1", apiRouter(s.db))

	port := cfg.App.Port
	if port == "" {
		port = "8080"
	}
	s.Server = http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	return s, nil
}

func NewStandaloneServer(cfg config.StandaloneApp, db database.Databaser) (*Server, error) {
	s := &Server{
		db:  db,
		cfg: cfg,
	}

	r := chi.NewMux().With(s.logRequest, injectAuth(db, cfg))

	r.Route("/api/v1", apiRouter(s.db))

	port := cfg.App.Port
	if port == "" {
		port = "8080"
	}
	s.Server = http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	return s, nil
}

func (s *Server) Run() error {
	slog.Info(fmt.Sprintf("Server listening on %s", s.Addr))
	if err := s.Server.ListenAndServe(); err != nil {
		return err
	}
	return nil
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
				userID := auth.GetUserID(cfg, r)
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

func getCtxValue(r *http.Request) ctxValue {
	ctx := r.Context()
	if v, ok := ctx.Value(ctxKey).(ctxValue); ok {
		return v
	}
	return ctxValue{}
}

func (c ctxValue) set(key string, val interface{}) ctxValue {
	c[key] = val
	return c
}

func (c ctxValue) get(key string) interface{} {
	return c[key]
}
