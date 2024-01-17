package server

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
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
	http.ServeFile(w, r, "./app/index.html")
}

func (s *Server) handleEntry(w http.ResponseWriter, r *http.Request) {
	dateString := r.FormValue("date")
	presence := r.FormValue("presence")
	note := r.FormValue("note")

	date, _ := time.Parse("2006-01-02", dateString)
	if date.IsZero() {
		date = time.Now()
	}

	e := db.Entry{
		Date:        date,
		CreatedDate: time.Now(),
		User:        auth.GetUserID(r),
		Presence:    presence,
		Reason:      note,
	}
	slog.Debug(fmt.Sprintf("%+v", e))

	id, err := s.db.SaveEntry(e)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to save entry: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}
	slog.Info(fmt.Sprintf("saved entry with id: %s", id))

	http.Redirect(w, r, "form", http.StatusSeeOther)
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

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	u := auth.GetUserID(r)

	summary, err := data.GenerateSummary(s.db, u)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to generate summary: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.ParseFiles("./app/summary.html"))
	if err := tmpl.Execute(w, summary); err != nil {
		slog.Error(fmt.Sprintf("failed to render summary: %v", err))
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
		return
	}
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

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "form", http.StatusTemporaryRedirect)
	})
	r.Get("/login", s.handleLogin)
	r.Get("/setup", s.handleSetup)
	r.Get("/notify", s.handleNotification)
	r.With(auth.Middleware).Get("/form", s.handleForm)
	r.With(auth.Middleware).Post("/submit", s.handleEntry)
	r.With(auth.Middleware).Get("/download", s.handleDownload)
	r.With(auth.Middleware).Get("/summary", s.handleSummary)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./app/static"))))
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
