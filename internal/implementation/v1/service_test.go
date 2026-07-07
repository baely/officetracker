package v1

import (
	"strings"
	"testing"
	"time"

	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/database/dbtest"
	"github.com/baely/officetracker/internal/report"
	"github.com/baely/officetracker/pkg/model"
)

// GetSettings decorates linked accounts with display names: known providers map
// to a friendly label, unknown ones are title-cased. It also normalises the
// calendar start month.
func TestGetSettingsDisplayNames(t *testing.T) {
	db := dbtest.New()
	db.LinkedAccounts = []model.LinkedAccount{
		{Provider: "github"},
		{Provider: "google-oauth2"},
		{Provider: "gitlab"}, // unknown -> title-cased
	}
	db.SaveCalendarPreferences(1, model.CalendarPreferences{TrackingYearStartMonth: 99}) // invalid
	svc := &Service{db: db}

	resp, err := svc.GetSettings(model.GetSettingsRequest{
		Meta: model.GetSettingsRequestMeta{UserID: 1},
	})
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}

	wantDisplay := map[string]string{"github": "GitHub", "google-oauth2": "Google", "gitlab": "Gitlab"}
	for _, acct := range resp.LinkedAccounts {
		if got := acct.ProviderDisplay; got != wantDisplay[acct.Provider] {
			t.Errorf("provider %q display = %q, want %q", acct.Provider, got, wantDisplay[acct.Provider])
		}
	}

	// 99 is out of range and must be normalised to the October default.
	if resp.CalendarPreferences.TrackingYearStartMonth != 10 {
		t.Errorf("calendar start month = %d, want normalised to 10", resp.CalendarPreferences.TrackingYearStartMonth)
	}
}

func TestGetSettingsLinkedAccountsError(t *testing.T) {
	db := dbtest.New()
	db.Errs = map[string]error{"GetUserLinkedAccounts": errInjected}
	svc := &Service{db: db}
	if _, err := svc.GetSettings(model.GetSettingsRequest{}); err == nil {
		t.Fatal("expected GetSettings to propagate linked-accounts error")
	}
}

// UpdateCalendarPreferences normalises the requested start month before saving,
// so an out-of-range value is stored as the default.
func TestUpdateCalendarPreferencesNormalises(t *testing.T) {
	db := dbtest.New()
	svc := &Service{db: db}

	_, err := svc.UpdateCalendarPreferences(model.UpdateCalendarPreferencesRequest{
		Meta: model.UpdateCalendarPreferencesRequestMeta{UserID: 1},
		Data: model.CalendarPreferences{TrackingYearStartMonth: 0}, // invalid
	})
	if err != nil {
		t.Fatalf("UpdateCalendarPreferences: %v", err)
	}

	stored, _ := db.GetCalendarPreferences(1)
	if stored.TrackingYearStartMonth != 10 {
		t.Errorf("stored start month = %d, want normalised 10", stored.TrackingYearStartMonth)
	}

	// A valid value passes through unchanged.
	svc.UpdateCalendarPreferences(model.UpdateCalendarPreferencesRequest{
		Meta: model.UpdateCalendarPreferencesRequestMeta{UserID: 1},
		Data: model.CalendarPreferences{TrackingYearStartMonth: 7},
	})
	stored, _ = db.GetCalendarPreferences(1)
	if stored.TrackingYearStartMonth != 7 {
		t.Errorf("stored start month = %d, want 7", stored.TrackingYearStartMonth)
	}
}

func TestUpdateThemeAndSchedulePreferences(t *testing.T) {
	db := dbtest.New()
	svc := &Service{db: db}

	_, err := svc.UpdateThemePreferences(model.UpdateThemePreferencesRequest{
		Meta: model.UpdateThemePreferencesRequestMeta{UserID: 1},
		Data: model.ThemePreferences{Theme: "dark", WeatherEnabled: true},
	})
	if err != nil {
		t.Fatalf("UpdateThemePreferences: %v", err)
	}
	if theme, _ := db.GetThemePreferences(1); theme.Theme != "dark" || !theme.WeatherEnabled {
		t.Errorf("theme not persisted: %+v", theme)
	}

	_, err = svc.UpdateSchedulePreferences(model.UpdateSchedulePreferencesRequest{
		Meta: model.UpdateSchedulePreferencesRequestMeta{UserID: 1},
		Data: model.SchedulePreferences{Monday: model.StateWorkFromOffice},
	})
	if err != nil {
		t.Fatalf("UpdateSchedulePreferences: %v", err)
	}
	if sched, _ := db.GetSchedulePreferences(1); sched.Monday != model.StateWorkFromOffice {
		t.Errorf("schedule not persisted: %+v", sched)
	}
}

// PostSecret rejects an empty/whitespace token name before generating anything.
func TestPostSecretEmptyName(t *testing.T) {
	db := dbtest.New()
	svc := &Service{db: db}

	for _, name := range []string{"", "   ", "\t"} {
		_, err := svc.PostSecret(model.PostSecretRequest{
			Meta: model.PostSecretRequestMeta{UserID: 1},
			Data: model.PostSecretRequestData{Name: name},
		})
		if err == nil {
			t.Errorf("PostSecret(%q) should reject empty name", name)
		}
	}
	if len(db.SavedSecrets) != 0 {
		t.Errorf("no secret should be saved for empty names, got %v", db.SavedSecrets)
	}
}

func TestPostSecretGeneratesAndStores(t *testing.T) {
	db := dbtest.New()
	svc := &Service{db: db}

	resp, err := svc.PostSecret(model.PostSecretRequest{
		Meta: model.PostSecretRequestMeta{UserID: 42},
		Data: model.PostSecretRequestData{Name: "  CI token  "},
	})
	if err != nil {
		t.Fatalf("PostSecret: %v", err)
	}
	if !strings.HasPrefix(resp.Secret, "officetracker:") {
		t.Errorf("secret = %q, want officetracker: prefix", resp.Secret)
	}
	if len(db.SavedSecrets) != 1 {
		t.Fatalf("expected 1 saved secret, got %d", len(db.SavedSecrets))
	}
	saved := db.SavedSecrets[0]
	if saved.UserID != 42 {
		t.Errorf("saved userID = %d, want 42", saved.UserID)
	}
	if saved.Name != "CI token" { // trimmed
		t.Errorf("saved name = %q, want trimmed \"CI token\"", saved.Name)
	}
	if saved.Secret != resp.Secret {
		t.Errorf("returned secret %q != stored secret %q", resp.Secret, saved.Secret)
	}
}

// ListTokens maps stored token metadata into the API shape, formatting the
// timestamp as RFC3339.
func TestListTokens(t *testing.T) {
	created := time.Date(2026, 7, 7, 9, 30, 0, 0, time.UTC)
	db := dbtest.New()
	db.Tokens = []database.TokenMetadata{
		{TokenID: 1, Name: "laptop", CreatedAt: created, Active: true},
		{TokenID: 2, Name: "ci", CreatedAt: created, Active: true},
	}
	svc := &Service{db: db}

	resp, err := svc.ListTokens(model.ListTokensRequest{
		Meta: model.ListTokensRequestMeta{UserID: 1},
	})
	if err != nil {
		t.Fatalf("ListTokens: %v", err)
	}
	if len(resp.Tokens) != 2 {
		t.Fatalf("got %d tokens, want 2", len(resp.Tokens))
	}
	if resp.Tokens[0].Name != "laptop" || resp.Tokens[0].TokenID != 1 {
		t.Errorf("token 0 = %+v", resp.Tokens[0])
	}
	if resp.Tokens[0].CreatedAt != created.Format(time.RFC3339) {
		t.Errorf("CreatedAt = %q, want %q", resp.Tokens[0].CreatedAt, created.Format(time.RFC3339))
	}
}

func TestRevokeToken(t *testing.T) {
	db := dbtest.New()
	svc := &Service{db: db}

	resp, err := svc.RevokeToken(model.RevokeTokenRequest{
		Meta: model.RevokeTokenRequestMeta{UserID: 1, TokenID: 9},
	})
	if err != nil || !resp.Success {
		t.Fatalf("RevokeToken = (%+v, %v), want success", resp, err)
	}
	if len(db.RevokedTokens) != 1 || db.RevokedTokens[0].TokenID != 9 {
		t.Errorf("revoke not recorded: %v", db.RevokedTokens)
	}

	// On DB error, Success is false and the error is returned.
	db.Errs = map[string]error{"RevokeToken": errInjected}
	resp, err = svc.RevokeToken(model.RevokeTokenRequest{
		Meta: model.RevokeTokenRequestMeta{UserID: 1, TokenID: 9},
	})
	if err == nil || resp.Success {
		t.Errorf("RevokeToken on error = (%+v, %v), want failure", resp, err)
	}
}

// GetStats returns the latest snapshot and formats ComputedAt as RFC3339 UTC,
// omitting it entirely when there is no snapshot.
func TestGetStats(t *testing.T) {
	db := dbtest.New()
	db.Snapshot = []model.StatWidget{{Key: "mau", Title: "MAU", Value: "10", Order: 1}}
	db.SetStatsTime(time.Date(2026, 7, 7, 1, 2, 3, 0, time.UTC))
	svc := &Service{db: db}

	resp, err := svc.GetStats(model.GetStatsRequest{})
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if len(resp.Widgets) != 1 || resp.Widgets[0].Key != "mau" {
		t.Errorf("widgets = %+v", resp.Widgets)
	}
	if resp.ComputedAt != "2026-07-07T01:02:03Z" {
		t.Errorf("ComputedAt = %q, want 2026-07-07T01:02:03Z", resp.ComputedAt)
	}
}

func TestGetStatsNoSnapshot(t *testing.T) {
	db := dbtest.New() // zero stats time
	svc := &Service{db: db}

	resp, err := svc.GetStats(model.GetStatsRequest{})
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if resp.ComputedAt != "" {
		t.Errorf("ComputedAt = %q, want empty when no snapshot", resp.ComputedAt)
	}
}

// GetReport / GetReportCSV wire the reporter through the tracking-year range and
// set the right content types.
func TestGetReportContentTypes(t *testing.T) {
	db := dbtest.New()
	db.SaveMonth(1, 1, 2024, model.MonthState{Days: map[int]model.DayState{
		2: {State: model.StateWorkFromOffice},
	}})
	svc := &Service{db: db, reporter: report.New(db)}

	pdf, err := svc.GetReport(model.GetReportRequest{
		Meta: model.GetReportRequestMeta{UserID: 1, Year: 2024},
		Name: "Alice",
	})
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}
	if pdf.ContentType != "application/pdf" {
		t.Errorf("PDF content type = %q", pdf.ContentType)
	}
	if b, ok := pdf.Data.([]byte); !ok || len(b) == 0 {
		t.Errorf("PDF data is empty or wrong type")
	}

	csv, err := svc.GetReportCSV(model.GetReportCSVRequest{
		Meta: model.GetReportCSVRequestMeta{UserID: 1, Year: 2024},
	})
	if err != nil {
		t.Fatalf("GetReportCSV: %v", err)
	}
	if csv.ContentType != "text/csv" {
		t.Errorf("CSV content type = %q", csv.ContentType)
	}
	if b, ok := csv.Data.([]byte); !ok || !strings.HasPrefix(string(b), "Date,State") {
		t.Errorf("CSV data unexpected: %v", csv.Data)
	}
}

func TestHealthAndValidateAuth(t *testing.T) {
	svc := &Service{}
	h, err := svc.Healthcheck(model.HealthCheckRequest{})
	if err != nil || h.Status != "ok" {
		t.Errorf("Healthcheck = (%+v, %v)", h, err)
	}
	v, err := svc.ValidateAuth(model.ValidateAuthRequest{})
	if err != nil || v.Status != "ok" {
		t.Errorf("ValidateAuth = (%+v, %v)", v, err)
	}
}
