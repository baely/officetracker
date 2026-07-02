package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/config"
	appctx "github.com/baely/officetracker/internal/context"
)

// requestWithAuthCtx builds a GET request whose context mirrors what
// injectAuth produces for the given auth method and optional user ID.
func requestWithAuthCtx(target string, method auth.Method, userID *int) *http.Request {
	val := appctx.CtxValue{}.Set(appctx.CtxAuthMethodKey, method)
	if userID != nil {
		val.Set(appctx.CtxUserIDKey, *userID)
	}
	r := httptest.NewRequest(http.MethodGet, target, nil)
	return r.WithContext(context.WithValue(r.Context(), appctx.CtxKey, val))
}

// TestHandleSettingsRedirectsUnauthenticated ensures visitors with no session
// are redirected to the hero page rather than served the settings page. The
// handler must bail out before touching the database, so a bare Server is
// enough — reaching further would panic on the nil service.
func TestHandleSettingsRedirectsUnauthenticated(t *testing.T) {
	s := &Server{cfg: config.IntegratedApp{}}
	zero := 0
	cases := map[string]*http.Request{
		// No cookie at all: injectAuth stores user ID 0 (MethodNone).
		"anonymous visitor": requestWithAuthCtx("/settings", auth.MethodNone, &zero),
		// Invalid/expired cookie: MethodSSO but no user ID in context.
		"invalid session": requestWithAuthCtx("/settings", auth.MethodSSO, nil),
	}
	for name, r := range cases {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			s.handleSettings(w, r)
			if w.Code != http.StatusTemporaryRedirect {
				t.Fatalf("expected %d, got %d", http.StatusTemporaryRedirect, w.Code)
			}
			if loc := w.Header().Get("Location"); loc != "/" {
				t.Fatalf("expected redirect to /, got %q", loc)
			}
		})
	}
}

// TestHandleDeveloperRedirectsNonSSO ensures non-SSO visitors are redirected
// and the developer page body is not written after the redirect headers.
func TestHandleDeveloperRedirectsNonSSO(t *testing.T) {
	s := &Server{}
	w := httptest.NewRecorder()
	s.handleDeveloper(w, requestWithAuthCtx("/developer", auth.MethodNone, nil))
	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected %d, got %d", http.StatusTemporaryRedirect, w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/" {
		t.Fatalf("expected redirect to /, got %q", loc)
	}
	if body := w.Body.String(); strings.Contains(body, "<html") {
		t.Fatalf("developer page rendered after redirect: %q", body)
	}
}

// TestHandleDeveloperServesSSOUsers ensures the SSO path still renders the page.
func TestHandleDeveloperServesSSOUsers(t *testing.T) {
	s := &Server{}
	w := httptest.NewRecorder()
	s.handleDeveloper(w, requestWithAuthCtx("/developer", auth.MethodSSO, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, w.Code)
	}
	if !strings.Contains(w.Body.String(), "Developer") {
		t.Fatalf("expected developer page content")
	}
}
