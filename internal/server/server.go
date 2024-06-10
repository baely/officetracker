package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/embed"
	v1 "github.com/baely/officetracker/internal/implementation/v1"
	"github.com/baely/officetracker/pkg/model"
)

type Server struct {
	http.Server
	cfg config.AppConfigurer
	db  database.Databaser

	// v1 implementation
	v1 model.Service
}

func NewServer(cfg config.AppConfigurer, db database.Databaser) (*Server, error) {
	s := &Server{
		db:  db,
		cfg: cfg,
		v1:  v1.New(db),
	}

	r := chi.NewMux().With(s.logRequest, injectAuth(db, cfg))

	// Form routes
	r.Get("/", s.handleIndex)
	r.Get("/{year:[0-9]{4}}-{month:[0-9]{1,2}}", s.handleForm)

	// API routes
	r.Route("/api/v1", apiRouter(s.v1))

	// Integrated app routes
	switch integratedCfg := cfg.(type) {
	case config.IntegratedApp:
		// Auth routes
		r.Route("/auth", auth.Router(integratedCfg, s.db))
		r.Get("/login", s.handleLogin)
		r.Get("/logout", s.handleLogout)
		// Cool stuff
		r.Get("/developer", s.handleDeveloper)
		// Boring stuff
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

	// TODO: remove
	chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		slog.Info(fmt.Sprintf("route: %s %s", method, route))
		return nil
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

// handleIndex handles the index route:
// - if the app is standalone or integrated and the user is logged in, it shows the form
// - if the app is integrated and the user is not logged in, it shows the hero
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	switch cfg := s.cfg.(type) {
	case config.StandaloneApp:
		s.handleForm(w, r)
		return
	case config.IntegratedApp:
		if auth.GetUserID(s.db, cfg, w, r) != 0 {
			s.handleForm(w, r)
			return
		} else {
			s.handleHero(w, r)
			return
		}
	}
	s.handleHero(w, r)
	return
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
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
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
		errorPage(w, err, "Invalid date", http.StatusBadRequest)
		return
	}

	yearlyData, err := s.v1.GetYear(model.GetYearRequest{
		Meta: model.GetYearRequestMeta{
			UserID: userID,
			Year:   year,
		},
	})
	if err != nil {
		err = fmt.Errorf("failed to get month data: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
		return
	}
	yearlyNotes, err := s.v1.GetNotes(model.GetNotesRequest{
		Meta: model.GetNotesRequestMeta{
			UserID: userID,
			Year:   year,
		},
	})
	if err != nil {
		err = fmt.Errorf("failed to get month note: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	yearlyDataByte, err := json.Marshal(yearlyData)
	if err != nil {
		err = fmt.Errorf("failed to marshal yearly data: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
		return
	}
	yearlyNotesByte, err := json.Marshal(yearlyNotes)
	if err != nil {
		err = fmt.Errorf("failed to marshal yearly notes: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
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
	ssoUri := auth.SSOUri(cfg)
	serveLogin(w, r, loginPage{
		SSOLink: ssoUri,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	auth.ClearCookie(w)
	slog.Info("logged out")
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (s *Server) handleDeveloper(w http.ResponseWriter, r *http.Request) {
	authMethod, err := getAuthMethod(r)
	if err != nil {
		err = fmt.Errorf("failed to get auth method: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
		return
	}
	if authMethod != AuthMethodSSO {
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

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	errorPage(w, nil, "Not found", http.StatusNotFound)
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
}
