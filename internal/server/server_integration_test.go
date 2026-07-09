package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database/dbtest"
	"github.com/baely/officetracker/internal/report"
	"github.com/baely/officetracker/pkg/model"
)

// newStandaloneServer builds a full standalone server (no Redis, no Auth0)
// wired to an in-memory database, and returns its HTTP handler. This exercises
// the real router, middleware chain, rate limiter and request/response
// plumbing end-to-end.
func newStandaloneServer(t *testing.T) (http.Handler, *dbtest.Fake) {
	t.Helper()
	db := dbtest.New()
	srv, err := NewServer(config.StandaloneApp{}, db, nil, report.New(db))
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return srv.Handler, db
}

func do(t *testing.T, h http.Handler, method, target, body string) *http.Response {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Result()
}

func bodyString(t *testing.T, res *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

func TestServerHealthCheck(t *testing.T) {
	h, _ := newStandaloneServer(t)
	res := do(t, h, http.MethodGet, "/api/v1/health/check", "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if b := bodyString(t, res); !strings.Contains(b, `"status":"ok"`) {
		t.Errorf("health body = %s", b)
	}
}

// The full write-then-read round-trip through the real HTTP API in standalone
// mode (which auto-authenticates as user 1).
func TestServerStateRoundTrip(t *testing.T) {
	h, _ := newStandaloneServer(t)

	if res := do(t, h, http.MethodPut, "/api/v1/state/2024/3/5", `{"data":{"state":2}}`); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT day status = %d, want 200", res.StatusCode)
	}

	res := do(t, h, http.MethodGet, "/api/v1/state/2024/3/5", "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET day status = %d", res.StatusCode)
	}
	if b := bodyString(t, res); !strings.Contains(b, `"state":2`) {
		t.Errorf("GET day body = %s, want state 2", b)
	}

	// Month view includes the day.
	res = do(t, h, http.MethodGet, "/api/v1/state/2024/3", "")
	if b := bodyString(t, res); !strings.Contains(b, `"state":2`) {
		t.Errorf("GET month body = %s", b)
	}

	// Year view is retrievable.
	if res := do(t, h, http.MethodGet, "/api/v1/state/2024", ""); res.StatusCode != http.StatusOK {
		t.Errorf("GET year status = %d", res.StatusCode)
	}
}

func TestServerNotesRoundTrip(t *testing.T) {
	h, _ := newStandaloneServer(t)

	if res := do(t, h, http.MethodPut, "/api/v1/note/2024/3", `{"data":{"note":"crunch"}}`); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT note status = %d", res.StatusCode)
	}
	res := do(t, h, http.MethodGet, "/api/v1/note/2024/3", "")
	if b := bodyString(t, res); !strings.Contains(b, "crunch") {
		t.Errorf("GET note body = %s", b)
	}
}

func TestServerSettingsRoundTrip(t *testing.T) {
	h, _ := newStandaloneServer(t)

	// Update the calendar start month via the API.
	if res := do(t, h, http.MethodPut, "/api/v1/settings/calendar", `{"data":{"tracking_year_start_month":7}}`); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT calendar status = %d", res.StatusCode)
	}
	// Fetch settings and confirm it round-trips.
	res := do(t, h, http.MethodGet, "/api/v1/settings", "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET settings status = %d", res.StatusCode)
	}
	if b := bodyString(t, res); !strings.Contains(b, `"tracking_year_start_month":7`) {
		t.Errorf("settings body = %s", b)
	}
}

func TestServerReportEndpoints(t *testing.T) {
	h, db := newStandaloneServer(t)
	db.SaveDay(1, 2, 1, 2024, model.DayState{State: model.StateWorkFromOffice})

	csv := do(t, h, http.MethodGet, "/api/v1/report/csv/2024-attendance", "")
	if csv.StatusCode != http.StatusOK {
		t.Fatalf("CSV status = %d", csv.StatusCode)
	}
	if ct := csv.Header.Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Errorf("CSV content-type = %q", ct)
	}
	if b := bodyString(t, csv); !strings.HasPrefix(b, "Date,State") {
		t.Errorf("CSV body = %q", b[:min(20, len(b))])
	}

	pdf := do(t, h, http.MethodGet, "/api/v1/report/pdf/2024-attendance", "")
	if pdf.StatusCode != http.StatusOK {
		t.Fatalf("PDF status = %d", pdf.StatusCode)
	}
	if ct := pdf.Header.Get("Content-Type"); !strings.Contains(ct, "application/pdf") {
		t.Errorf("PDF content-type = %q", ct)
	}
}

func TestServerStatsEndpoint(t *testing.T) {
	h, _ := newStandaloneServer(t)
	res := do(t, h, http.MethodGet, "/api/v1/stats", "")
	if res.StatusCode != http.StatusOK {
		t.Errorf("stats status = %d, want 200 (public)", res.StatusCode)
	}
}

// Developer endpoints require SSO/Secret; a standalone (Excluded) session is
// rejected with 401.
func TestServerDeveloperEndpointsRejectExcluded(t *testing.T) {
	h, _ := newStandaloneServer(t)
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/developer/tokens", ""},
		{http.MethodPost, "/api/v1/developer/secret", `{"data":{"name":"x"}}`},
	} {
		res := do(t, h, tc.method, tc.path, tc.body)
		if res.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want 401", tc.method, tc.path, res.StatusCode)
		}
	}
}

// /health/auth requires MethodSecret; standalone (Excluded) is rejected.
func TestServerHealthAuthRejectsExcluded(t *testing.T) {
	h, _ := newStandaloneServer(t)
	res := do(t, h, http.MethodGet, "/api/v1/health/auth", "")
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("health/auth status = %d, want 401", res.StatusCode)
	}
}

func TestServerAPINotFound(t *testing.T) {
	h, _ := newStandaloneServer(t)
	res := do(t, h, http.MethodGet, "/api/v1/does-not-exist", "")
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("unknown api route status = %d, want 404", res.StatusCode)
	}
}

// The index is now a static, cacheable page (the redirect to the current month
// for authenticated users happens client-side).
func TestServerIndexStaticAndCacheable(t *testing.T) {
	h, _ := newStandaloneServer(t)
	res := do(t, h, http.MethodGet, "/", "")
	if res.StatusCode != http.StatusOK {
		t.Errorf("index status = %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("index content-type = %q, want html", ct)
	}
	if cc := res.Header.Get("Cache-Control"); !strings.Contains(cc, "public") {
		t.Errorf("index Cache-Control = %q, want a public (cacheable) directive", cc)
	}
}

// HTML pages render in standalone mode and are served with public, cacheable
// headers so Firebase can cache them.
func TestServerHTMLPages(t *testing.T) {
	h, _ := newStandaloneServer(t)
	for _, path := range []string{"/2024-03", "/settings", "/stats"} {
		res := do(t, h, http.MethodGet, path, "")
		if res.StatusCode != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", path, res.StatusCode)
		}
		if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Errorf("GET %s content-type = %q, want html", path, ct)
		}
		if cc := res.Header.Get("Cache-Control"); !strings.Contains(cc, "public") {
			t.Errorf("GET %s Cache-Control = %q, want public", path, cc)
		}
	}
}

// The auth-context bootstrap endpoint reports the viewer's auth state and is not
// cached. In standalone mode the single local user is authenticated.
func TestServerContextEndpoint(t *testing.T) {
	h, _ := newStandaloneServer(t)
	res := do(t, h, http.MethodGet, "/api/v1/context", "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("context status = %d, want 200", res.StatusCode)
	}
	b := bodyString(t, res)
	if !strings.Contains(b, `"authenticated":true`) || !strings.Contains(b, `"standalone":true`) {
		t.Errorf("context body = %s, want authenticated+standalone", b)
	}
	if cc := res.Header.Get("Cache-Control"); !strings.Contains(cc, "no-store") {
		t.Errorf("context Cache-Control = %q, want no-store", cc)
	}
}

func TestServerStaticAssets(t *testing.T) {
	h, _ := newStandaloneServer(t)
	res := do(t, h, http.MethodGet, "/favicon.ico", "")
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "image/png" {
		t.Errorf("favicon = %d %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
	res = do(t, h, http.MethodGet, "/static/themes.css", "")
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "text/css" {
		t.Errorf("themes.css = %d %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
}
