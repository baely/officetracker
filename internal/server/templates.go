package server

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/baely/officetracker/internal/embed"
)

type formPage struct {
	YearlyState template.JS
	YearlyNotes template.JS
}

func serveForm(w http.ResponseWriter, r *http.Request, page formPage) {
	if err := embed.Form.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute form template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type heroPage struct {
}

func serveHero(w http.ResponseWriter, r *http.Request, page heroPage) {
	http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)

	// TODO: create proper hero
	//if err := embed.Hero.Execute(w, page); err != nil {
	//	err = fmt.Errorf("failed to execute hero template: %w", err)
	//	errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	//}
}

type loginPage struct {
	SSOLink string
}

func serveLogin(w http.ResponseWriter, r *http.Request, page loginPage) {
	if err := embed.Login.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute login template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type settingsPage struct{}

func serveSettings(w http.ResponseWriter, r *http.Request, page settingsPage) {
	if err := embed.Settings.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute settings template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type developerPage struct{}

func serveDeveloper(w http.ResponseWriter, r *http.Request, page developerPage) {
	if err := embed.Developer.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute developer template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type tosPage struct {
}

func serveTos(w http.ResponseWriter, r *http.Request, page tosPage) {
	if err := embed.Tos.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute tos template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type privacyPage struct {
}

func servePrivacy(w http.ResponseWriter, r *http.Request, page privacyPage) {
	if err := embed.Privacy.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute privacy template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type ErrorPage struct {
	ErrorMessage string
}

func errorPage(w http.ResponseWriter, err error, userMsg string, status int) {
	slog.Error(err.Error())
	if err := embed.Error.Execute(w, ErrorPage{
		ErrorMessage: err.Error(),
	}); err != nil {
		err = fmt.Errorf("failed to execute error template: %w", err)
		slog.Error(err.Error())
	}
}
