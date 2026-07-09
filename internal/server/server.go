package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/embed"
	v1 "github.com/baely/officetracker/internal/implementation/v1"
	"github.com/baely/officetracker/internal/report"
)

type Server struct {
	http.Server
	cfg   config.AppConfigurer
	db    database.Databaser
	redis *database.Redis
	auth  *auth.Auth

	// v1 implementation
	v1 *v1.Service
}

func NewServer(cfg config.AppConfigurer, db database.Databaser, redis *database.Redis, reporter report.Reporter) (*Server, error) {
	s := &Server{
		db:    db,
		redis: redis,
		cfg:   cfg,
		v1:    v1.New(db, reporter),
	}

	author, err := auth.NewAuth(cfg, db, redis)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize auth: %w", err)
	}
	s.auth = author

	limiter := newRateLimiter(redis, authedRateLimits, unauthedRateLimits)
	r := chi.NewMux().With(injectAuth(db, cfg), s.logRequest, limiter.middleware)

	// Static, cacheable HTML pages. These carry no per-user data; the client
	// fetches everything dynamic from /api and enforces auth-based redirects and
	// nav rendering itself (see internal/embed/html/bases/base.html).
	//
	// Suspension page (must be accessible to suspended users)
	r.Get("/suspended", staticPage(pageSuspended))

	// Form routes
	r.Get("/", staticPage(pageIndex))
	r.Get("/{year:[0-9]{4}}-{month:[0-9]{1,2}}", staticPage(pageForm))

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Uncached auth-state bootstrap for the static frontend. Public: an
		// anonymous caller gets an all-false context.
		r.Get("/context", handleContext)
		// Handlers needing the auth service live here rather than apiRouter.
		r.With(AllowedAuthMethods(auth.MethodSSO, auth.MethodSecret)).
			Get("/account/link", s.handleAccountLinkURL)
		r.With(AllowedAuthMethods(auth.MethodSSO, auth.MethodSecret, auth.MethodExcluded)).
			Post("/auth/logout", s.handleLogoutToken)
		apiRouter(s.v1)(r)
	})

	r.Route("/mcp/v1", func(r chi.Router) {
		mcpRouter(s.v1)(r)
	})

	// Settings available in both standalone and integrated modes
	r.Get("/settings", staticPage(pageSettings))

	// Public stats dashboard (unauthenticated, aggregate-only).
	r.Get("/stats", staticPage(pageStats))

	// Integrated app routes
	switch integratedCfg := cfg.(type) {
	case config.IntegratedApp:
		// Auth routes
		r.Route("/auth", auth.Router(integratedCfg, s.db, s.auth))
		r.Get("/login", staticPage(pageLogin))
		r.Get("/logout", s.handleLogout)
		// Cool stuff
		r.Get("/developer", staticPage(pageDeveloper))
		// Boring stuff
		r.Get("/tos", staticPage(pageTos))
		r.Get("/privacy", staticPage(pagePrivacy))
	}

	r.Route("/static", staticHandler)
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
		w.Write(embed.OfficeBuilding)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		serveErrorPage(w, http.StatusNotFound)
	})

	port := cfg.GetApp().Port
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

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfg.(config.IntegratedApp)
	auth.ClearCookie(cfg, w)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// handleAccountLinkURL returns an Auth0 account-linking URL for the signed-in
// user (expires after 10 minutes).
func (s *Server) handleAccountLinkURL(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		writeError(w, "account linking is not available", http.StatusNotImplemented)
		return
	}
	userID, err := getUserID(r)
	if err != nil || userID == 0 {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	url, err := s.auth.GenerateAuth0AuthLink(userID)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to generate account link url: %v", err))
		writeError(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}
	b, err := json.Marshal(map[string]string{"url": url})
	if err != nil {
		writeError(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// handleLogoutToken revokes the API token presented on this request. No-op for
// standalone and cookie/SSO sessions.
func (s *Server) handleLogoutToken(w http.ResponseWriter, r *http.Request) {
	cfg, ok := s.cfg.(config.IntegratedApp)
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}
	secret, method := auth.GetAuth(cfg, r)
	if method == auth.MethodSecret && secret != "" {
		if err := s.db.RevokeSecretByValue(secret); err != nil {
			slog.Error(fmt.Sprintf("failed to revoke token on logout: %v", err))
			writeError(w, internalErrorMsg, http.StatusInternalServerError)
			return
		}
		slog.Info("revoked token on logout")
	}
	w.WriteHeader(http.StatusOK)
}

func staticHandler(r chi.Router) {
	r.Get("/github-mark-white.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
		w.Write(embed.GitHubMark)
	})
	r.Get("/office-building.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
		w.Write(embed.OfficeBuilding)
	})
	r.Get("/themes.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
		w.Write(embed.ThemesCSS)
	})
	r.Get("/skyline.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
		w.Write(embed.SkylineSVG)
	})
}
