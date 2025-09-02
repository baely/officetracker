package server

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/embed"
	"github.com/baely/officetracker/pkg/model"
)

type basePage struct {
	IsLoggedIn    bool
	IsStandalone  bool
}

type formPage struct {
	basePage
	YearlyState template.JS
	YearlyNotes template.JS
}

func serveForm(w http.ResponseWriter, r *http.Request, page formPage) {
	page.basePage = getBasePageData(r)
	if err := embed.Form.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute form template: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type heroPage struct {
	basePage
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
	basePage
	SSOLink string
}

func serveLogin(w http.ResponseWriter, r *http.Request, page loginPage) {
	page.basePage = getBasePageData(r)
	if err := embed.Login.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute login template: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type settingsPage struct {
	basePage
	GithubAccounts      []string
	GithubAuthURL       string
	ThemePreferences    model.ThemePreferences
	SchedulePreferences model.SchedulePreferences
}

func serveSettings(w http.ResponseWriter, r *http.Request, page settingsPage) {
	page.basePage = getBasePageData(r)
	if err := embed.Settings.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute settings template: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type developerPage struct{
	basePage
}

func serveDeveloper(w http.ResponseWriter, r *http.Request, page developerPage) {
	page.basePage = getBasePageData(r)
	if err := embed.Developer.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute developer template: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type tosPage struct {
	basePage
}

func serveTos(w http.ResponseWriter, r *http.Request, page tosPage) {
	page.basePage = getBasePageData(r)
	if err := embed.Tos.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute tos template: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type privacyPage struct {
	basePage
}

func servePrivacy(w http.ResponseWriter, r *http.Request, page privacyPage) {
	page.basePage = getBasePageData(r)
	if err := embed.Privacy.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute privacy template: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type suspendedPage struct {
	basePage
}

func serveSuspended(w http.ResponseWriter, r *http.Request, page suspendedPage) {
	page.basePage = getBasePageData(r)
	if err := embed.Suspended.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute suspended template: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type ErrorPage struct {
	basePage
	ErrorMessage string
}

func getBasePageData(r *http.Request) basePage {
	authMethod, _ := getAuthMethod(r)
	isLoggedIn := authMethod == auth.MethodSSO || authMethod == auth.MethodSecret || authMethod == auth.MethodExcluded
	isStandalone := authMethod == auth.MethodExcluded
	return basePage{
		IsLoggedIn:   isLoggedIn,
		IsStandalone: isStandalone,
	}
}

func errorPage(w http.ResponseWriter, r *http.Request, err error, userMsg string, status int) {
	slog.Error(err.Error())
	w.WriteHeader(status)
	if err := embed.Error.Execute(w, ErrorPage{
		basePage:     getBasePageData(r),
		ErrorMessage: err.Error(),
	}); err != nil {
		err = fmt.Errorf("failed to execute error template: %w", err)
		slog.Error(err.Error())
	}
}
