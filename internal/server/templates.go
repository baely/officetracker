package server

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/embed"
	"github.com/baely/officetracker/internal/util"
	"github.com/baely/officetracker/pkg/model"
)

type basePage struct {
	IsLoggedIn   bool
	IsStandalone bool
}

type formPage struct {
	basePage
	YearlyState        template.JS
	YearlyNotes        template.JS
	TrackingStartMonth int
	TargetPercent      int
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
	page.basePage = getBasePageData(r)
	if err := embed.Hero.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute hero template: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
	}
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
	LinkedAccounts      []model.LinkedAccount
	Auth0AuthURL        string
	ThemePreferences    model.ThemePreferences
	SchedulePreferences model.SchedulePreferences
	CalendarPreferences model.CalendarPreferences
	TargetPreferences   model.TargetPreferences
}

func serveSettings(w http.ResponseWriter, r *http.Request, page settingsPage) {
	page.basePage = getBasePageData(r)
	if err := embed.Settings.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute settings template: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

type developerPage struct {
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

type reportRow struct {
	Month   string
	Present int
	Total   int
	Percent string
}

type reportPage struct {
	basePage
	Year     int
	Rows     []reportRow
	Headline string
}

func serveReport(w http.ResponseWriter, r *http.Request, page reportPage) {
	page.basePage = getBasePageData(r)
	if err := embed.Report.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute report template: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

// buildReportSummary computes the per-month attendance breakdown for a tracking
// year, mirroring how the form page's summary counted days: "present" is office
// days (actual + scheduled), "total" is all work days (WFH + office, actual +
// scheduled). Months with no work days are omitted.
func buildReportSummary(state model.YearState, year, startMonth int) ([]reportRow, string) {
	startMonth = util.NormaliseStartMonth(startMonth)
	firstYear, secondYear := util.TrackingYearCalendarYears(year, startMonth)

	var rows []reportRow
	var totalPresent, totalDays int
	for offset := 0; offset < 12; offset++ {
		month := (startMonth-1+offset)%12 + 1
		monthYear := secondYear
		if month >= startMonth {
			monthYear = firstYear
		}

		var present, total int
		for _, day := range state.Months[month].Days {
			switch day.State {
			case model.StateWorkFromOffice, model.StateScheduledWorkFromOffice:
				present++
				total++
			case model.StateWorkFromHome, model.StateScheduledWorkFromHome:
				total++
			}
		}
		if total == 0 {
			continue
		}
		totalPresent += present
		totalDays += total
		rows = append(rows, reportRow{
			Month:   fmt.Sprintf("%s %d", time.Month(month).String(), monthYear),
			Present: present,
			Total:   total,
			Percent: fmt.Sprintf("%.2f%%", float64(present)/float64(total)*100),
		})
	}

	var percent float64
	if totalDays > 0 {
		percent = float64(totalPresent) / float64(totalDays) * 100
	}
	headline := fmt.Sprintf("Present in office for %d out of %d days. (%.2f%%)", totalPresent, totalDays, percent)
	return rows, headline
}

type statWidgetGroup struct {
	Name    string
	Widgets []model.StatWidget
}

type statsPage struct {
	basePage
	Groups      []statWidgetGroup
	LastUpdated string
}

func serveStats(w http.ResponseWriter, r *http.Request, page statsPage) {
	page.basePage = getBasePageData(r)
	if err := embed.Stats.Execute(w, page); err != nil {
		err = fmt.Errorf("failed to execute stats template: %w", err)
		errorPage(w, r, err, internalErrorMsg, http.StatusInternalServerError)
	}
}

// groupStatWidgets clusters widgets by their Group field, preserving first-seen
// order for both groups and the widgets within them.
func groupStatWidgets(widgets []model.StatWidget) []statWidgetGroup {
	var groups []statWidgetGroup
	index := make(map[string]int)
	for _, wgt := range widgets {
		i, ok := index[wgt.Group]
		if !ok {
			index[wgt.Group] = len(groups)
			groups = append(groups, statWidgetGroup{Name: wgt.Group})
			i = len(groups) - 1
		}
		groups[i].Widgets = append(groups[i].Widgets, wgt)
	}
	return groups
}

// formatLastUpdated renders the snapshot timestamp for display, or a fallback
// when no snapshot exists yet.
func formatLastUpdated(computedAt string) string {
	if computedAt == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, computedAt)
	if err != nil {
		return computedAt
	}
	// Render in Melbourne time. The container runs in UTC, so t.Local() would
	// show UTC; fall back to UTC only if the tz database is unavailable.
	loc, err := time.LoadLocation("Australia/Melbourne")
	if err != nil {
		loc = time.UTC
	}
	return t.In(loc).Format("2 Jan 2006, 3:04 PM MST")
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
	// err may be nil (e.g. a 404 from handleNotFound); fall back to userMsg.
	errMsg := userMsg
	if err != nil {
		slog.Error(err.Error())
		errMsg = err.Error()
	} else {
		slog.Error(userMsg)
	}
	w.WriteHeader(status)
	if err := embed.Error.Execute(w, ErrorPage{
		basePage:     getBasePageData(r),
		ErrorMessage: errMsg,
	}); err != nil {
		err = fmt.Errorf("failed to execute error template: %w", err)
		slog.Error(err.Error())
	}
}
