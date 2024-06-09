package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/data"
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/embed"
	"github.com/baely/officetracker/internal/models"
)

const (
	internalErrorMsg = "Internal server error"
)

type Server struct {
	http.Server
	cfg config.AppConfigurer
	db  database.Databaser
}

type submission struct {
	Month string      `json:"month"`
	Year  string      `json:"year"`
	Days  map[int]int `json:"days"`
	Notes string      `json:"notes"`
}

type response struct {
	State []int  `json:"state"`
	Notes string `json:"notes"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfg.(config.IntegratedApp)
	if err := embed.Login.Execute(w, struct{ SSOLink string }{auth.SSOUri(cfg)}); err != nil {
		slog.Error(fmt.Sprintf("failed to render login: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	auth.ClearCookie(w)
	http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	w.Write(embed.Setup)
}

func (s *Server) handleForm(w http.ResponseWriter, r *http.Request) {
	month := chi.URLParam(r, "month")

	if month == "setup" || month == "download" {
		http.Redirect(w, r, fmt.Sprintf("/%s", month), http.StatusTemporaryRedirect)
	}

	if month == "" {
		month = time.Now().Format("2006-01")
		http.Redirect(w, r, fmt.Sprintf("/form/%s", month), http.StatusTemporaryRedirect)
	}
	t, err := time.Parse("2006-01", month)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to parse month: %v", err))
		http.Error(w, "bad date part", http.StatusBadRequest)
		return
	}

	sum, err := data.GenerateSummary(s.db, s.getUserID(r), int(t.Month()), t.Year())
	if err != nil {
		slog.Error(fmt.Sprintf("failed to generate summary: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	respSum := make(map[string]map[int]int)
	for _, month := range sum.MonthData {
		parts := strings.Split(month.MonthUri, "/")
		yearPart := parts[1]
		monthPart := parts[2]
		key := fmt.Sprintf("%s-%02s", yearPart, monthPart)

		if _, ok := respSum[key]; !ok {
			respSum[key] = make(map[int]int)
		}
		respSum[key][1] = month.TotalDays - month.TotalPresent
		respSum[key][2] = month.TotalPresent
	}
	jsonSum, err := json.Marshal(respSum)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to marshal summary: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	entry, err := s.db.GetEntries(s.getUserID(r), int(t.Month()), t.Year())
	if err != nil {
		slog.Error(fmt.Sprintf("failed to get entries: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}
	state := make([]string, 32)
	for i := 0; i < 32; i++ {
		state[i] = "0"
	}
	for day, dayState := range entry.Days {
		dd, _ := strconv.Atoi(day)
		state[dd] = fmt.Sprintf("%d", dayState)
	}
	stateStr := template.JS("[" + strings.Join(state, ",") + "]")

	if err = embed.Index.Execute(w, struct {
		Summary template.JS
		State   template.JS
		Notes   string
	}{Summary: template.JS(jsonSum), State: stateStr, Notes: entry.Notes}); err != nil {
		slog.Error(fmt.Sprintf("failed to render form: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	month := chi.URLParam(r, "month")

	t, err := time.Parse("2006-01", month)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to parse month: %v", err))
		http.Error(w, "bad date part", http.StatusBadRequest)
		return
	}

	entry, err := s.db.GetEntries(s.getUserID(r), int(t.Month()), t.Year())
	if err != nil {
		slog.Error(fmt.Sprintf("failed to get entries: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	resp := response{
		State: make([]int, 32),
		Notes: entry.Notes,
	}

	for day, dayState := range entry.Days {
		dd, _ := strconv.Atoi(day)
		resp.State[dd] = dayState
	}

	b, err := json.Marshal(resp)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to marshal state: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (s *Server) handleEntry(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to read request body: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	var sub submission
	err = json.Unmarshal(b, &sub)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to unmarshal request body: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	month, err := strconv.Atoi(sub.Month)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to parse month: %v", err))
		http.Error(w, "bad date part", http.StatusBadRequest)
		return
	}
	year, err := strconv.Atoi(sub.Year)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to parse year: %v", err))
		http.Error(w, "bad date part", http.StatusBadRequest)
		return
	}

	days := make(map[string]int)

	for day, state := range sub.Days {
		days[fmt.Sprintf("%d", day)] = state
	}

	e := models.Entry{
		User:       s.getUserID(r),
		CreateDate: time.Now(),
		Month:      month,
		Year:       year,
		Days:       days,
		Notes:      sub.Notes,
	}
	if err = s.db.SaveEntry(e); err != nil {
		slog.Error(fmt.Sprintf("failed to save entry: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	w.Write([]byte("OK"))
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	u := s.getUserID(r)

	b, err := data.GenerateCsv(s.db, u)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to generate excel: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=officecheck.csv")
	w.Write(b)
}

func (s *Server) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(fmt.Sprintf("request: %s %s", r.Method, r.URL.Path))
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info(fmt.Sprintf("request: %s %s took %s", r.Method, r.URL.Path, time.Since(start)))
	})
}

func (s *Server) getUserID(r *http.Request) string {
	switch cfg := s.cfg.(type) {
	case config.IntegratedApp:
		return auth.GetUserID(cfg, r)
	case config.StandaloneApp:
		return ""
	default:
		return ""
	}
}

func NewServer(cfg config.IntegratedApp, db database.Databaser) (*Server, error) {
	s := &Server{
		db:  db,
		cfg: cfg,
	}

	r := chi.NewMux().With(s.logRequest)

	// Anonymous routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "form", http.StatusTemporaryRedirect)
	})
	r.Get("/login", s.handleLogin)
	r.Get("/logout", s.handleLogout)

	// User routes
	r.With(auth.Middleware(cfg, s.db)).Get("/form", s.handleForm)
	r.With(auth.Middleware(cfg, s.db)).Get("/form/{month}", s.handleForm)
	r.With(auth.Middleware(cfg, s.db)).Get("/user-state/{month}", s.handleState)
	r.With(auth.Middleware(cfg, s.db)).Post("/submit", s.handleEntry)
	r.With(auth.Middleware(cfg, s.db)).Get("/setup", s.handleSetup)
	r.With(auth.Middleware(cfg, s.db)).Get("/download", s.handleDownload)

	// Static routes
	r.Handle("/static/github-mark-white.png", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(embed.GitHubMark)
	}))

	// Subroutes
	r.Route("/auth", auth.Router(cfg, s.db))

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

	r := chi.NewMux().With(s.logRequest)

	// Anonymous routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "form", http.StatusTemporaryRedirect)
	})

	// User routes
	r.Get("/form", s.handleForm)
	r.Get("/form/{month}", s.handleForm)
	r.Get("/user-state/{month}", s.handleState)
	r.Post("/submit", s.handleEntry)
	r.Get("/download", s.handleDownload)

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
