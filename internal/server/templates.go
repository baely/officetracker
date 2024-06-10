package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/baely/officetracker/internal/embed"
	"github.com/baely/officetracker/pkg/model"
)

func serveForm(w http.ResponseWriter, r *http.Request, monthState model.MonthState, monthNote model.Note) {
	if err := embed.Form.Execute(w, struct {
		MonthState model.MonthState
		MonthNote  model.Note
	}{
		MonthState: monthState,
		MonthNote:  monthNote,
	}); err != nil {
		err = fmt.Errorf("failed to execute form template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

func serveHero(w http.ResponseWriter, r *http.Request) {
	if err := embed.Hero.Execute(w, nil); err != nil {
		err = fmt.Errorf("failed to execute hero template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

func serveLogin(w http.ResponseWriter, r *http.Request, ssoLink string) {
	if err := embed.Login.Execute(w, struct {
		SSOLink string
	}{
		SSOLink: ssoLink,
	}); err != nil {
		err = fmt.Errorf("failed to execute login template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

func serveTos(w http.ResponseWriter, r *http.Request) {
	if err := embed.Tos.Execute(w, nil); err != nil {
		err = fmt.Errorf("failed to execute tos template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

func servePrivacy(w http.ResponseWriter, r *http.Request) {
	if err := embed.Privacy.Execute(w, nil); err != nil {
		err = fmt.Errorf("failed to execute privacy template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

func errorPage(w http.ResponseWriter, err error, userMsg string, status int) {
	slog.Error(err.Error())
	http.Error(w, userMsg, status)
}
