package server

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/baely/officetracker/internal/embed"
	"github.com/baely/officetracker/internal/theme"
)

type formPage struct {
	YearlyState template.JS
	YearlyNotes template.JS
	Theme       template.CSS
}

func serveForm(w http.ResponseWriter, r *http.Request, page formPage) {
	// Add the theme if not already set
	if page.Theme == "" {
		page.Theme = theme.GetSeasonalTheme().ToCSS()
	}
	
	if err := embed.Form.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute form template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type heroPage struct {
	Theme template.CSS
}

func serveHero(w http.ResponseWriter, r *http.Request, page heroPage) {
	// Add the theme if not already set
	if page.Theme == "" {
		page.Theme = theme.GetSeasonalTheme().ToCSS()
	}
	
	http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)

	// TODO: create proper hero
	//if err := embed.Hero.Execute(w, page); err != nil {
	//	err = fmt.Errorf("failed to execute hero template: %w", err)
	//	errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	//}
}

type loginPage struct {
	SSOLink string
	Theme   template.CSS
}

func serveLogin(w http.ResponseWriter, r *http.Request, page loginPage) {
	// Add the theme if not already set
	if page.Theme == "" {
		page.Theme = theme.GetSeasonalTheme().ToCSS()
	}
	
	if err := embed.Login.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute login template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type settingsPage struct {
	GithubAccounts []string
	GithubAuthURL  string
	Theme          template.CSS
}

func serveSettings(w http.ResponseWriter, r *http.Request, page settingsPage) {
	// Add the theme if not already set
	if page.Theme == "" {
		page.Theme = theme.GetSeasonalTheme().ToCSS()
	}
	
	if err := embed.Settings.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute settings template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type developerPage struct{
	Theme template.CSS
}

func serveDeveloper(w http.ResponseWriter, r *http.Request, page developerPage) {
	// Add the theme if not already set
	if page.Theme == "" {
		page.Theme = theme.GetSeasonalTheme().ToCSS()
	}
	
	if err := embed.Developer.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute developer template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type tosPage struct {
	Theme template.CSS
}

func serveTos(w http.ResponseWriter, r *http.Request, page tosPage) {
	// Add the theme if not already set
	if page.Theme == "" {
		page.Theme = theme.GetSeasonalTheme().ToCSS()
	}
	
	if err := embed.Tos.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute tos template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type privacyPage struct {
	Theme template.CSS
}

func servePrivacy(w http.ResponseWriter, r *http.Request, page privacyPage) {
	// Add the theme if not already set
	if page.Theme == "" {
		page.Theme = theme.GetSeasonalTheme().ToCSS()
	}
	
	if err := embed.Privacy.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute privacy template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type ErrorPage struct {
	ErrorMessage string
	Theme        template.CSS
}

func errorPage(w http.ResponseWriter, err error, userMsg string, status int) {
	slog.Error(err.Error())
	page := ErrorPage{
		ErrorMessage: err.Error(),
		Theme: theme.GetSeasonalTheme().ToCSS(),
	}
	
	if err := embed.Error.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute error template: %w", err)
		slog.Error(err.Error())
	}
}
