package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/embed"
	v1 "github.com/baely/officetracker/internal/implementation/v1"
	"github.com/baely/officetracker/internal/report"
	"github.com/baely/officetracker/pkg/model"
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

	author, err := auth.NewAuth(cfg, redis)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize auth: %w", err)
	}
	s.auth = author

	r := chi.NewMux().With(Otel, injectAuth(db, cfg), s.logRequest)

	// Suspension page (must be accessible to suspended users)
	r.Get("/suspended", s.handleSuspended)

	// Form routes (protected by suspension check)
	r.With(checkSuspension(db)).Get("/", s.handleIndex)
	r.With(checkSuspension(db)).Get("/{year:[0-9]{4}}-{month:[0-9]{1,2}}", s.handleForm)

	// API routes (protected by suspension check)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(checkSuspension(db))
		apiRouter(s.v1)(r)
	})

	r.Route("/mcp/v1", func(r chi.Router) {
		r.Use(checkSuspension(db))
		mcpRouter(s.v1)(r)
	})

	// Settings available in both standalone and integrated modes
	r.With(checkSuspension(db)).Get("/settings", s.handleSettings)

	// Integrated app routes
	switch integratedCfg := cfg.(type) {
	case config.IntegratedApp:
		// Auth routes (not protected by suspension check to allow login/logout)
		r.Route("/auth", auth.Router(integratedCfg, s.db, s.redis, s.auth))
		r.Get("/login", s.handleLogin)
		r.Get("/logout", s.handleLogout)
		// Cool stuff (protected by suspension check)
		r.With(checkSuspension(db)).Get("/developer", s.handleDeveloper)
		// Boring stuff (not protected by suspension check)
		r.Get("/tos", s.handleTos)
		r.Get("/privacy", s.handlePrivacy)
	}

	r.Route("/static", staticHandler)
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
		w.Write(embed.OfficeBuilding)
	})

	r.NotFound(s.handleNotFound)

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

// handleIndex handles the index route:
// - if the app is standalone or integrated and the user is logged in, it shows the form
// - if the app is integrated and the user is not logged in, it shows the hero
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	switch s.cfg.(type) {
	case config.StandaloneApp:
		s.handleForm(w, r)
		return
	case config.IntegratedApp:
		method, _ := getAuthMethod(r)
		var loggedInMethods = []auth.Method{auth.MethodSSO, auth.MethodSecret}
		if !slices.Contains(loggedInMethods, method) {
			s.handleHero(w, r)
		}

		s.handleForm(w, r)
	default:
		s.handleHero(w, r)
	}
}

func (s *Server) handleForm(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if errors.Is(err, ErrNoUserInCtx) || userID == 0 {
		slog.Info("no user id in context, redirecting to login")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	if err != nil {
		err = fmt.Errorf("failed to get user id: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	yearStr := chi.URLParam(r, "year")
	monthStr := chi.URLParam(r, "month")
	if yearStr == "" || monthStr == "" {
		http.Redirect(w, r, fmt.Sprintf("/%s", time.Now().Format("2006-01")), http.StatusTemporaryRedirect)
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		err = fmt.Errorf("failed to convert year to int: %w", err)
		errorPage(w, r, err, "Invalid date", http.StatusBadRequest)
		return
	}
	month, err := strconv.Atoi(monthStr)
	if err != nil {
		err = fmt.Errorf("failed to convert month to int: %w", err)
		errorPage(w, r, err, "Invalid date", http.StatusBadRequest)
		return
	}

	if month > 9 {
		year++
	}

	yearlyData, err := s.v1.GetYear(model.GetYearRequest{
		Meta: model.GetYearRequestMeta{
			UserID: userID,
			Year:   year,
		},
	})
	if err != nil {
		err = fmt.Errorf("failed to get year data: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
		return
	}
	yearlyNotes, err := s.v1.GetNotes(model.GetNotesRequest{
		Meta: model.GetNotesRequestMeta{
			UserID: userID,
			Year:   year,
		},
	})
	if err != nil {
		err = fmt.Errorf("failed to get year note: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	yearlyDataByte, err := json.Marshal(yearlyData)
	if err != nil {
		err = fmt.Errorf("failed to marshal yearly data: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
		return
	}
	yearlyNotesByte, err := json.Marshal(yearlyNotes)
	if err != nil {
		err = fmt.Errorf("failed to marshal yearly notes: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
		return
	}
	yearlyDataStr := string(yearlyDataByte)
	yearlyNotesStr := string(yearlyNotesByte)

	serveForm(w, r, formPage{
		YearlyState: template.JS(yearlyDataStr),
		YearlyNotes: template.JS(yearlyNotesStr),
	})
}

func (s *Server) handleHero(w http.ResponseWriter, r *http.Request) {
	serveHero(w, r, heroPage{})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfg.(config.IntegratedApp)

	var ssoUri string
	var err error
	if getDebug(r) {
		ssoUri, err = s.auth.Auth0SSOUri()
	} else {
		ssoUri, err = auth.SSOUri(cfg, s.redis)
	}

	if err != nil {
		slog.Error(fmt.Sprintf("failed to generate SSO URI: %v", err))
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
		return
	}
	serveLogin(w, r, loginPage{
		SSOLink: ssoUri,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfg.(config.IntegratedApp)
	auth.ClearCookie(cfg, w)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		err = fmt.Errorf("failed to get user id: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
		return
	}
	settings, err := s.v1.GetSettings(model.GetSettingsRequest{
		Meta: model.GetSettingsRequestMeta{
			UserID: userID,
		},
	})
	if err != nil {
		err = fmt.Errorf("failed to get settings: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	// Handle GitHub auth only for integrated mode
	var authURL string
	var githubAccounts []string
	switch cfg := s.cfg.(type) {
	case config.IntegratedApp:
		authURL, err = auth.GenerateGitHubAuthLink(r.Context(), cfg, s.redis, userID)
		if err != nil {
			errorPage(w, r, fmt.Errorf("failed to generate github auth link: %v", err), internalErrorMsg, http.StatusInternalServerError)
			return
		}
		githubAccounts = settings.GithubAccounts
	default:
		// Standalone mode - no GitHub integration
		authURL = ""
		githubAccounts = []string{}
	}

	serveSettings(w, r, settingsPage{
		GithubAccounts:      githubAccounts,
		GithubAuthURL:       authURL,
		ThemePreferences:    settings.ThemePreferences,
		SchedulePreferences: settings.SchedulePreferences,
	})
}

func (s *Server) handleDeveloper(w http.ResponseWriter, r *http.Request) {
	authMethod, err := getAuthMethod(r)
	if err != nil {
		err = fmt.Errorf("failed to get auth method: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
		return
	}
	if authMethod != auth.MethodSSO {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}

	serveDeveloper(w, r, developerPage{})
}

func (s *Server) handleTos(w http.ResponseWriter, r *http.Request) {
	serveTos(w, r, tosPage{})
}

func (s *Server) handlePrivacy(w http.ResponseWriter, r *http.Request) {
	servePrivacy(w, r, privacyPage{})
}

func (s *Server) handleSuspended(w http.ResponseWriter, r *http.Request) {
	serveSuspended(w, r, suspendedPage{})
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	errorPage(w, r, nil, "Not found", http.StatusNotFound)
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
}
