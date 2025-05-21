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
	Theme       embed.Theme // Added Theme field
}

// serveForm is now likely superseded by renderThemeTemplate in server.go for themed pages.
// If a default/non-themed version is still needed, this could be kept and used.
// For clarity, I'll comment it out if it's not directly used by themed rendering.
/*
func serveForm(w http.ResponseWriter, r *http.Request, page formPage) {
	// Decide which template to execute based on whether page.Theme is populated
	// or if there's a global default.
	// This logic is now primarily in server.go's handleForm.
	tmplToExecute := embed.Form // Fallback to old form
	if page.Theme.Index != nil {
		// This path shouldn't be hit if server.go's renderThemeTemplate is used correctly.
		// tmplToExecute = page.Theme.Index // This would be wrong if page.Theme not set
		slog.Warn("serveForm called directly with a theme; should use renderThemeTemplate")
	}
	if err := tmplToExecute.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute form template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}
*/

type heroPage struct {
	Theme embed.Theme // Added Theme field, if hero pages become themeable
}

// serveHero might also be superseded or need adjustment if hero pages are themed.
// The current redirect logic in server.go's handleHero might mean this isn't used often.
/*
func serveHero(w http.ResponseWriter, r *http.Request, page heroPage) {
	http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)

	// TODO: create proper hero
	//if err := embed.Hero.Execute(w, page); err != nil {
	//	err = fmt.Errorf("failed to execute hero template: %w", err)
	//	errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	//}
}
*/

type loginPage struct {
	SSOLink string
	Theme   embed.Theme // Added Theme field
}

// serveLogin is also likely superseded by renderThemeTemplate in server.go.
/*
func serveLogin(w http.ResponseWriter, r *http.Request, page loginPage) {
	// Logic similar to serveForm, now handled in server.go's handleLogin
	tmplToExecute := embed.Login // Fallback to old login
	if page.Theme.Login != nil {
		slog.Warn("serveLogin called directly with a theme; should use renderThemeTemplate")
	}
	if err := tmplToExecute.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute login template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}
*/

type settingsPage struct {
	GithubAccounts  []string
	GithubAuthURL   string
	Theme           embed.Theme // Theme for the settings page itself (if themed)
	UserTheme       string      // User's current theme preference
	AvailableThemes map[string]embed.Theme
	AppConfig       config.App // To access DefaultTheme in template
}

// serveSettings will be used, but the template it renders (`embed.Settings`)
// will need to be updated to use the Theme field for asset paths and theme selection.
func serveSettings(w http.ResponseWriter, r *http.Request, page settingsPage) {
	// The actual template execution is now expected to happen in server.go's handleSettings
	// using a generic render method, or this function needs to be updated to choose
	// the correct settings template if settings pages themselves become themed.
	// For now, assuming embed.Settings is a global template that can handle theme data.
	if err := embed.Settings.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute settings template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type developerPage struct{
	Theme embed.Theme // If developer page needs theme awareness
}

// serveDeveloper might be superseded or need adjustment.
func serveDeveloper(w http.ResponseWriter, r *http.Request, page developerPage) {
	if err := embed.Developer.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute developer template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type tosPage struct {
	Theme embed.Theme // If ToS page needs theme awareness
}

// serveTos might be superseded or need adjustment.
func serveTos(w http.ResponseWriter, r *http.Request, page tosPage) {
	if err := embed.Tos.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute tos template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type privacyPage struct {
	Theme embed.Theme // If privacy page needs theme awareness
}

// servePrivacy might be superseded or need adjustment.
func servePrivacy(w http.ResponseWriter, r *http.Request, page privacyPage) {
	if err := embed.Privacy.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute privacy template: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type ErrorPage struct {
	ErrorMessage string
	Theme        embed.Theme // If error page needs theme awareness
}

// errorPage should ideally also use renderThemeTemplate if errors are to be themed.
// For now, it uses the global embed.Error.
func errorPage(w http.ResponseWriter, err error, userMsg string, status int) {
	// In a fully themed app, you might try to get the user's current theme
	// or a default theme to render the error page.
	// currentTheme, _ := embed.GetTheme("default") // Or some context-aware theme
	slog.Error(fmt.Sprintf("User-facing error: %s (Status: %d)", userMsg, status), "internal_error", err)
	w.WriteHeader(status) // Ensure status is set before writing body

	// Prepare data for the error template
	// pageData := ErrorPage{ErrorMessage: userMsg} // Simplified error message
	// If 'err' is not nil and we want to show its details (dev mode?), use err.Error()
	// For production, userMsg is safer. Here, we're logging err.Error() and showing userMsg.
	
	// For now, using the existing global error template.
	// If embed.Error should be themed, its rendering needs to change.
	if renderErr := embed.Error.Execute(w, ErrorPage{
		ErrorMessage: userMsg, // Show user-friendly message
		// Theme: currentTheme, // Pass theme if error template is theme-aware
	}); renderErr != nil {
		// Fallback if error template itself fails
		slog.Error("failed to execute error template", "execution_error", renderErr)
		http.Error(w, "An unexpected error occurred and the error page could not be displayed.", http.StatusInternalServerError)
	}
}
