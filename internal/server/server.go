package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/embed"
	v1 "github.com/baely/officetracker/internal/implementation/v1"
	"github.com/baely/officetracker/internal/report"
	"github.com/baely/officetracker/pkg/model"
)

type Server struct {
	http.Server
	cfg   config.AppConfigurer
	db    database.Databaser
	redis *database.Redis

	// v1 implementation
	v1 *v1.Service
}

func NewServer(cfg config.AppConfigurer, db database.Databaser, redis *database.Redis, reporter report.Reporter) (*Server, error) {
	s := &Server{
		db:    db,
		redis: redis,
		cfg:   cfg,
		v1:    v1.New(db, reporter, cfg), // Pass cfg to v1.New
	}

	r := chi.NewMux().With(Otel, injectAuth(db, cfg), s.logRequest)

	// Form routes
	r.Get("/", s.handleIndex)
	r.Get("/{year:[0-9]{4}}-{month:[0-9]{1,2}}", s.handleForm)

	// API routes
	r.Route("/api/v1", apiRouter(s.v1))

	// Integrated app routes
	switch integratedCfg := cfg.(type) {
	case config.IntegratedApp:
		// Auth routes
		r.Route("/auth", auth.Router(integratedCfg, s.db, s.redis))
		r.Get("/login", s.handleLogin)
		r.Get("/logout", s.handleLogout)
		// Cool stuff
		r.Get("/settings", s.handleSettings)
		r.Get("/developer", s.handleDeveloper)
		// Boring stuff
		r.Get("/tos", s.handleTos)
		r.Get("/privacy", s.handlePrivacy)
	}

	r.Route("/static", staticRouter)
	r.Route("/theme", themeStaticRouter) // New route for theme assets

	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
		w.Write(embed.OfficeBuilding)
	})

	r.NotFound(s.handleNotFound)

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
	switch s.cfg.(type) {
	case config.StandaloneApp:
		s.handleForm(w, r)
		return
	case config.IntegratedApp:
		// Get user's theme preference (this will be implemented later)
		// For now, let's assume "infinite_city" or a default theme.
		// userThemeName, _ := s.getUserThemePreference(userID) // Placeholder
		userThemeName := "infinite_city" // Hardcoded for now, will be dynamic
		selectedTheme, themeExists := embed.GetTheme(userThemeName)
		if !themeExists {
			// Fallback to a default theme if the preferred one doesn't exist
			// selectedTheme, _ = embed.GetTheme("default") // Assuming "default" theme is set up
			// For now, if infinite_city is missing, it's an error or use existing hero.
			slog.Error("Selected theme not found", "theme", userThemeName)
			s.handleHero(w, r) // Or render an error
			return
		}

		method, _ := getAuthMethod(r)
		var loggedInMethods = []auth.Method{auth.MethodSSO, auth.MethodSecret}
		if !slices.Contains(loggedInMethods, method) {
			// For non-loggedIn users, show the theme's login page or a hero page
			// If the theme has a specific hero/landing page, use that.
			// For now, we'll assume login page is the entry for themed experience.
			// If theme has no specific login, could redirect to generic /login or show generic hero.
			s.renderThemeTemplate(w, r, selectedTheme.Login, loginPage{ // Assuming loginPage struct is appropriate
				SSOLink: auth.SSOUri(s.cfg.(config.IntegratedApp)),
				Theme:   selectedTheme,
			})
			return
		}
		// If logged in, proceed to the theme's index/form page
		s.handleForm(w, r, selectedTheme) // Pass the theme to handleForm

	default:
		// Fallback for other config types or if theme logic isn't fully integrated yet.
		s.handleHero(w, r)
	}
}

func (s *Server) handleForm(w http.ResponseWriter, r *http.Request, theme ...embed.Theme) {
	userID, err := getUserID(r)
	if errors.Is(err, ErrNoUserInCtx) || userID == 0 {
		// If no user, and it's an integrated app, redirect to themed login or main page.
		if _, ok := s.cfg.(config.IntegratedApp); ok {
			// Determine theme (default or from some other context if applicable)
			// For now, let's assume we need to redirect to the main page which handles login display.
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			slog.Info("no user id in context for handleForm, redirecting to /")
			return
		}
		// For standalone or other cases, might be an error or different handling.
		// For now, treating as error if user is expected.
		errorPage(w, errors.New("user ID missing in context for form view"), "Authentication required.", http.StatusUnauthorized)
		return
	}
	if err != nil {
		err = fmt.Errorf("failed to get user id: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	// Determine which theme to use
	var currentTheme embed.Theme
	if len(theme) > 0 {
		currentTheme = theme[0]
	} else {
		// Fallback or default theme logic if not passed (e.g. for standalone app)
		// userThemeName, _ := s.getUserThemePreference(userID) // Placeholder
		userThemeName := "infinite_city" // Default to infinite_city if not specified
		var themeExists bool
		currentTheme, themeExists = embed.GetTheme(userThemeName)
		if !themeExists {
			slog.Error("Default theme for form not found", "theme", userThemeName)
			errorPage(w, fmt.Errorf("theme %s not found", userThemeName), internalErrorMsg, http.StatusInternalServerError)
			return
		}
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
	month, err := strconv.Atoi(monthStr)
	if err != nil {
		err = fmt.Errorf("failed to convert month to int: %w", err)
		errorPage(w, err, "Invalid date", http.StatusBadRequest)
		return
	}

	if month > 9 {
		year++
	}

	yearlyData, err := s.v1.GetYear(model.GetYearRequest{
		Meta: model.GetYearRequestMeta{
			UserID: userID,
			Year:   year,
		},
	})
	if err != nil {
		err = fmt.Errorf("failed to get year data: %w", err)
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
		err = fmt.Errorf("failed to get year note: %w", err)
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

	// Use currentTheme.Index to render
	s.renderThemeTemplate(w, r, currentTheme.Index, formPage{
		YearlyState: template.JS(yearlyDataStr),
		YearlyNotes: template.JS(yearlyNotesStr),
		Theme:       currentTheme, // Pass theme for template to use (e.g., for static asset paths)
	})
}

func (s *Server) handleHero(w http.ResponseWriter, r *http.Request) {
	// This might need to be theme-aware too, or be a very generic landing.
	// For now, assume default theme or a generic hero if no theme context.
	// If we want themed hero pages, this would need similar logic to handleIndex.
	// userThemeName := "default" // Or determine dynamically
	// selectedTheme, themeExists := embed.GetTheme(userThemeName)
	// if !themeExists { ... error ... }
	// s.renderThemeTemplate(w, r, selectedTheme.Hero, heroPage{Theme: selectedTheme})
	serveHero(w, r, heroPage{}) // Keeping original for now
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfg.(config.IntegratedApp)
	ssoUri := auth.SSOUri(cfg)

	// Determine theme (e.g., default or from a query param, or system default)
	// For now, let's use "infinite_city" as the example.
	// In a real scenario, this might be the system's default login theme.
	loginThemeName := "infinite_city" // Or s.cfg.GetDefaultTheme()
	selectedTheme, themeExists := embed.GetTheme(loginThemeName)
	if !themeExists {
		slog.Error("Login theme not found", "theme", loginThemeName)
		// Fallback to old login page or an error
		serveLogin(w, r, loginPage{SSOLink: ssoUri}) // Original function
		return
	}

	s.renderThemeTemplate(w, r, selectedTheme.Login, loginPage{
		SSOLink: ssoUri,
		Theme:   selectedTheme, // Pass theme for template to use
	})
}

// renderThemeTemplate is a helper to execute a theme's template.
func (s *Server) renderThemeTemplate(w http.ResponseWriter, r *http.Request, tmpl *template.Template, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := tmpl.Execute(w, data)
	if err != nil {
		slog.Error("failed to execute theme template", "error", err)
		// Fallback to a generic error page, or the old error page function
		http.Error(w, internalErrorMsg, http.StatusInternalServerError)
	}
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cfg := s.cfg.(config.IntegratedApp)
	auth.ClearCookie(cfg, w)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		err = fmt.Errorf("failed to get user id: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
		return
	}
	settings, err := s.v1.GetSettings(model.GetSettingsRequest{
		Meta: model.GetSettingsRequestMeta{
			UserID: userID,
		},
	})
	if err != nil {
		err = fmt.Errorf("failed to get settings: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
		return
	}

	cfg := s.cfg.(config.IntegratedApp)
	authURL, err := auth.GenerateGitHubAuthLink(r.Context(), cfg, s.redis, userID)
	if err != nil {
		errorPage(w, fmt.Errorf("failed to generate github auth link: %v", err), internalErrorMsg, http.StatusInternalServerError)
		return
	}

	// Pass available themes and user's current theme to the template
	pageData := settingsPage{
		GithubAccounts: settings.GithubAccounts,
		GithubAuthURL:  authURL,
		UserTheme:      settings.Theme, // From v1.GetSettings
		AvailableThemes: embed.AvailableThemes,
		AppConfig:      s.cfg.GetApp(), // Pass App config
		// Theme: This field in settingsPage might be for the page's own theme, not user's preference for other pages.
		// If settings page itself is themed, assign appropriately:
		// Theme: currentTheme, (where currentTheme is determined like in handleIndex/handleForm)
	}

	// If settings page is themed, select its theme (e.g. user's current theme or system default)
	// For now, assuming settings page uses a default or is not heavily themed by 'infinite_city' style itself.
	// If it were, you'd do:
	// settingsPageTheme, _ := s.getUserThemePreferenceOrDefault(userID)
	// pageData.Theme = settingsPageTheme

	serveSettings(w, r, pageData)
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		errorPage(w, fmt.Errorf("failed to get user id for update settings: %w", err), "Authentication required", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		errorPage(w, fmt.Errorf("failed to parse settings form: %w", err), "Invalid request", http.StatusBadRequest)
		return
	}

	selectedTheme := r.FormValue("theme")
	if selectedTheme == "" {
		errorPage(w, fmt.Errorf("theme value missing from form"), "Theme selection is required", http.StatusBadRequest)
		return
	}

	// Call the v1 service to update settings
	_, err = s.v1.UpdateSettings(model.PutSettingsRequest{
		Meta: model.PutSettingsRequestMeta{UserID: userID},
		Data: model.PutSettingsData{Theme: selectedTheme},
	})

	if err != nil {
		// Log the detailed error
		slog.Error("failed to update settings", "user_id", userID, "error", err)
		// Show a generic error to the user
		// It might be good to repopulate the settings page with an error message
		// For now, redirecting back to settings, or could show an error page.
		// To show error on settings page, handleSettings would need to accept an error message param.
		errorPage(w, err, "Failed to update settings. Please try again.", http.StatusInternalServerError)
		return
	}

	// Redirect back to settings page (or show a success message)
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}


func (s *Server) handleDeveloper(w http.ResponseWriter, r *http.Request) {
	authMethod, err := getAuthMethod(r)
	if err != nil {
		err = fmt.Errorf("failed to get auth method: %w", err)
		errorPage(w, err, internalErrorMsg, http.StatusInternalServerError)
		return
	}
	if authMethod != auth.MethodSSO {
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

func staticRouter(r chi.Router) {
	// Serves global static files (like those directly in embed/static)
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
	// Add other global static assets here if any
}

func themeStaticRouter(r chi.Router) {
	// Serves static assets from themes, e.g., /theme/infinite_city/style.css
	r.Get("/{themeName}/*", func(w http.ResponseWriter, r *http.Request) {
		themeName := chi.URLParam(r, "themeName")
		theme, ok := embed.GetTheme(themeName)
		if !ok {
			http.NotFound(w, r)
			return
		}

		// The path to the asset within the theme's static FS
		// chi.URLParam(r, "*") gives the path after {themeName}/
		assetPath := chi.URLParam(r, "*")

		// Ensure http.FileServer uses the theme's StaticFS
		// We need to strip the theme name from the path for the FileServer,
		// as it expects paths relative to its root.
		// However, StaticFS is already a sub-FS for the theme, so assetPath is correct.
		http.FileServer(http.FS(theme.StaticFS)).ServeHTTP(w, r)
	})
}

// Placeholder for fetching user theme preference - to be implemented fully later
// func (s *Server) getUserThemePreference(userID int64) (string, error) {
// 	// TODO: Query database for user's theme preference
// 	// For now, return a default or hardcoded value
// 	if userID > 0 { // only if user is identified
//		// settings, err := s.v1.GetSettings(model.GetSettingsRequest{Meta: model.GetSettingsRequestMeta{UserID: userID}})
//		// if err == nil && settings.Theme != "" {
//		// return settings.Theme, nil
//		// }
//	 }
//	 // return s.cfg.GetDefaultTheme(), nil // Or a hardcoded default like "default"
// 	return "infinite_city", nil // Hardcoded for now
// }
