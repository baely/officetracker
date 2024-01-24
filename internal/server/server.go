package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/data"
	db "github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/integration"
)

const (
	internalErrorMsg = "Internal server error"
)

type Server struct {
	http.Server
	db *db.Client
}

type submission struct {
	Day   string `json:"day"`
	Month string `json:"month"`
	Year  string `json:"year"`
	State string `json:"state"`
}

func (s *Server) handleNotification(w http.ResponseWriter, r *http.Request) {
	backendEndpoint := os.Getenv("BACKEND_ENDPOINT")
	p := integration.NewPayload("Office Check", "Are you in the office today?", backendEndpoint)
	if err := p.Send(); err != nil {
		slog.Error(fmt.Sprintf("failed to send notification: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	w.Write([]byte("OK"))
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("./app/login.html"))
	if err := tmpl.Execute(w, struct{ SSOLink string }{auth.GitHubAuthUri()}); err != nil {
		slog.Error(fmt.Sprintf("failed to render login: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./app/setup.html")
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

	summary, err := data.GenerateSummary(s.db, auth.GetUserID(r), int(t.Month()), t.Year())
	if err != nil {
		slog.Error(fmt.Sprintf("failed to generate summary: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.ParseFiles("./app/picker.html"))
	if err := tmpl.Execute(w, struct{ Summary data.Summary }{Summary: summary}); err != nil {
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

	entries, err := s.db.GetEntries(auth.GetUserID(r), int(t.Month()), t.Year())
	if err != nil {
		slog.Error(fmt.Sprintf("failed to get entries: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	state := make([]int, 32)
	for _, e := range entries {
		state[e.Day] = e.State
	}

	b, err := json.Marshal(state)
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

	day, err := strconv.Atoi(sub.Day)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to parse day: %v", err))
		http.Error(w, "bad date part", http.StatusBadRequest)
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
	state, err := strconv.Atoi(sub.State)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to parse state: %v", err))
		http.Error(w, "bad state", http.StatusBadRequest)
		return
	}

	e := db.Entry{
		User:       auth.GetUserID(r),
		CreateDate: time.Now(),
		Day:        day,
		Month:      month,
		Year:       year,
		State:      state,
	}
	id, err := s.db.SaveEntry(e)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to save entry: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	slog.Info(fmt.Sprintf("saved entry with id: %s", id))
	w.Write([]byte("OK"))
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserID(r)

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
		next.ServeHTTP(w, r)
	})
}

func NewServer(port string) (*Server, error) {
	s := &Server{}

	r := chi.NewMux().With(s.logRequest)

	// Anonymous routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "form", http.StatusTemporaryRedirect)
	})
	r.Get("/login", s.handleLogin)
	r.Get("/notify", s.handleNotification)

	// User routes
	r.With(auth.Middleware).Get("/form", s.handleForm)
	r.With(auth.Middleware).Get("/form/{month}", s.handleForm)
	r.With(auth.Middleware).Get("/user-state/{month}", s.handleState)
	r.With(auth.Middleware).Post("/submit", s.handleEntry)
	r.With(auth.Middleware).Get("/setup", s.handleSetup)
	r.With(auth.Middleware).Get("/download", s.handleDownload)

	// Static routes
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./app/static"))))

	// Subroutes
	r.Route("/auth", auth.Router())

	s.Server = http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	var err error
	s.db, err = db.NewFirestoreClient()
	if err != nil {
		return nil, err
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
