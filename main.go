package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/go-chi/chi/v5"
)

type Entry struct {
	Date     time.Time
	Presence string
	Reason   string
}

type server struct {
	http.Server
	db *firestore.Client
}

func (s *server) handleNotification(w http.ResponseWriter, r *http.Request) {
	p := NewPayload("Office Check", "Are you in the office today?")
	slog.Info(fmt.Sprintf("request: %s %s", r.Method, r.URL.Path))
	p.AddAction("Log it", backendEndpoint)
	err := p.Send()
	if err != nil {
		slog.Error(fmt.Sprintf("failed to send notification: %v", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("OK"))
}

func (s *server) handleForm(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "/app/index.html")
}

func (s *server) handleEntry(w http.ResponseWriter, r *http.Request) {
	presence := r.FormValue("presence")
	note := r.FormValue("note")

	e := Entry{
		Date:     time.Now(),
		Presence: presence,
		Reason:   note,
	}

	slog.Info(fmt.Sprintf("%+v", e))

	err := s.saveEntry(e)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to save entry: %v", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/form", http.StatusSeeOther)
}

func (s *server) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info(fmt.Sprintf("request: %s %s", r.Method, r.URL.Path))
		next.ServeHTTP(w, r)
	})
}

func (s *server) saveEntry(e Entry) error {
	ctx := context.Background()
	_, _, err := s.db.Collection("entries").Add(ctx, e)
	if err != nil {
		return fmt.Errorf("failed to save entry: %v", err)
	}
	return nil
}

func newServer(port string) *server {
	s := &server{}

	r := chi.NewMux().With(s.logRequest)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/form", http.StatusTemporaryRedirect)
	})
	r.Get("/notify", s.handleNotification)
	r.Get("/form", s.handleForm)
	r.Post("/submit", s.handleEntry)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	})

	s.Server = http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	s.db = newFirestoreClient()

	return s
}

func (s *server) run() error {
	slog.Info(fmt.Sprintf("server listening on %s", s.Addr))
	if err := s.Server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	s := newServer(port)
	if err := s.run(); err != nil {
		panic(err)
	}
}
