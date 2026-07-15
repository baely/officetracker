package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/config"
	context2 "github.com/baely/officetracker/internal/context"
)

func requestWithAuthMethod(m auth.Method) *http.Request {
	val := context2.CtxValue{}
	val.Set(context2.CtxAuthMethodKey, m)
	r := httptest.NewRequest("GET", "/", nil)
	return r.WithContext(context.WithValue(r.Context(), context2.CtxKey, val))
}

// AllowedAuthMethods lets a request through only when its auth method is in the
// allow-list; otherwise it short-circuits with 401 (or 500 if the method is
// missing from context entirely).
func TestAllowedAuthMethods(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	t.Run("allowed method passes", func(t *testing.T) {
		nextCalled = false
		w := httptest.NewRecorder()
		AllowedAuthMethods(auth.MethodSSO, auth.MethodSecret)(next).ServeHTTP(w, requestWithAuthMethod(auth.MethodSSO))
		if !nextCalled || w.Code != http.StatusOK {
			t.Errorf("allowed request: nextCalled=%v code=%d", nextCalled, w.Code)
		}
	})

	t.Run("disallowed method is 401", func(t *testing.T) {
		nextCalled = false
		w := httptest.NewRecorder()
		AllowedAuthMethods(auth.MethodSSO)(next).ServeHTTP(w, requestWithAuthMethod(auth.MethodNone))
		if nextCalled {
			t.Error("next should not be called for a disallowed method")
		}
		if w.Code != http.StatusUnauthorized {
			t.Errorf("code = %d, want 401", w.Code)
		}
	})

	t.Run("missing auth method is 500", func(t *testing.T) {
		nextCalled = false
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil) // no ctx auth method
		AllowedAuthMethods(auth.MethodSSO)(next).ServeHTTP(w, r)
		if nextCalled {
			t.Error("next should not be called when auth method is absent")
		}
		if w.Code != http.StatusInternalServerError {
			t.Errorf("code = %d, want 500", w.Code)
		}
	})
}

// In standalone mode every request is treated as the single local user (id 1)
// with the "excluded" auth method.
func TestInjectAuthStandalone(t *testing.T) {
	var gotUserID int
	var gotMethod auth.Method
	var gotOK bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, _ = getUserID(r)
		gotMethod, gotOK = mustAuthMethod(r)
	})

	cfg := config.StandaloneApp{}
	w := httptest.NewRecorder()
	injectAuth(cfg, nil)(next).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if gotUserID != 1 {
		t.Errorf("standalone userID = %d, want 1", gotUserID)
	}
	if !gotOK || gotMethod != auth.MethodExcluded {
		t.Errorf("standalone auth method = %v, want excluded", gotMethod)
	}
}

// In integrated mode with no credentials, the request resolves to MethodNone
// and no private Cache-Control header is set.
func TestInjectAuthIntegratedAnonymous(t *testing.T) {
	var gotMethod auth.Method
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, _ = mustAuthMethod(r)
	})

	cfg := config.IntegratedApp{Domain: config.Domain{Domain: "officetracker.com.au"}}
	w := httptest.NewRecorder()
	injectAuth(cfg, nil)(next).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if gotMethod != auth.MethodNone {
		t.Errorf("anonymous integrated method = %v, want none", gotMethod)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "" {
		t.Errorf("anonymous request should not set Cache-Control, got %q", cc)
	}
}

// fakeResolver stands in for *auth.Auth in middleware tests.
type fakeResolver struct {
	uid int
	err error
}

func (f fakeResolver) GetUserID(context.Context, string, auth.Method) (int, error) {
	return f.uid, f.err
}

// A cookie carrying an invalid session marks the request as SSO but fails user
// resolution: the cookie is cleared, no user id is placed in context, and the
// private Cache-Control header is set.
func TestInjectAuthIntegratedInvalidCookie(t *testing.T) {
	var hadUser bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := getUserID(r)
		hadUser = err == nil
	})

	cfg := config.IntegratedApp{
		App:    config.App{Env: "cloud"},
		Domain: config.Domain{Domain: "officetracker.com.au"},
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "__session", Value: "not-a-valid-session"})
	w := httptest.NewRecorder()
	injectAuth(cfg, fakeResolver{err: errors.New("session invalid")})(next).ServeHTTP(w, r)

	if hadUser {
		t.Error("invalid session should not resolve a user id into context")
	}
	if cc := w.Header().Get("Cache-Control"); cc != "private, no-store" {
		t.Errorf("Cache-Control = %q, want private, no-store", cc)
	}
	// The bad cookie should be cleared.
	var cleared bool
	for _, c := range w.Result().Cookies() {
		if c.Name == "__session" && c.Value == "" {
			cleared = true
		}
	}
	if !cleared {
		t.Error("invalid session cookie should be cleared")
	}
}

// A valid session cookie resolves to its user id.
func TestInjectAuthIntegratedValidCookie(t *testing.T) {
	var gotUserID int
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, _ = getUserID(r)
	})

	cfg := config.IntegratedApp{
		App:    config.App{Env: "cloud"},
		Domain: config.Domain{Domain: "officetracker.com.au"},
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "__session", Value: "opaque-session-id"})
	w := httptest.NewRecorder()
	injectAuth(cfg, fakeResolver{uid: 42})(next).ServeHTTP(w, r)

	if gotUserID != 42 {
		t.Errorf("resolved userID = %d, want 42", gotUserID)
	}
}

// mustAuthMethod reads the auth method from context for assertions.
func mustAuthMethod(r *http.Request) (auth.Method, bool) {
	m, ok := context2.GetCtxValue(r).Get(context2.CtxAuthMethodKey).(auth.Method)
	return m, ok
}
