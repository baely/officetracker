package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database/dbtest"
)

func testCfg() config.IntegratedApp {
	return config.IntegratedApp{
		App:    config.App{Env: "cloud"},
		Domain: config.Domain{Domain: "officetracker.com.au"},
	}
}

func TestCookieName(t *testing.T) {
	cases := []struct {
		env  string
		want string
	}{
		{"", "user"},
		{"cloud", "user"},
		{"dev", "user_dev"},
		{"staging", "user_staging"},
	}
	for _, c := range cases {
		cfg := config.IntegratedApp{App: config.App{Env: c.env}}
		if got := cookieName(cfg); got != c.want {
			t.Errorf("cookieName(env=%q) = %q, want %q", c.env, got, c.want)
		}
	}
}

func TestLegacyCookieName(t *testing.T) {
	cases := []struct {
		env  string
		want string
	}{
		{"", "__session"},
		{"cloud", "__session"},
		{"dev", "__session_dev"},
		{"staging", "__session_staging"},
	}
	for _, c := range cases {
		cfg := config.IntegratedApp{App: config.App{Env: c.env}}
		if got := legacyCookieName(cfg); got != c.want {
			t.Errorf("legacyCookieName(env=%q) = %q, want %q", c.env, got, c.want)
		}
	}
}

func TestValidateDevSecret(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"Bearer abc123", "abc123"},
		{"bearer abc123", "abc123"}, // case-insensitive prefix
		{"BEARER abc123", "abc123"},
		{"Basic abc123", ""}, // wrong scheme
		{"abc123", ""},       // no scheme
		{"Bearer ", ""},      // empty token after prefix
	}
	for _, c := range cases {
		if got := validateDevSecret(c.in); got != c.want {
			t.Errorf("validateDevSecret(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestGetAuth(t *testing.T) {
	cfg := testCfg()

	t.Run("cookie -> SSO", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{Name: cookieName(cfg), Value: "the-session-id"})
		tok, method := GetAuth(cfg, r)
		if tok != "the-session-id" || method != MethodSSO {
			t.Errorf("got (%q, %v), want (the-session-id, SSO)", tok, method)
		}
	})

	t.Run("authorization header -> Secret", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Authorization", "Bearer my-secret")
		tok, method := GetAuth(cfg, r)
		if tok != "my-secret" || method != MethodSecret {
			t.Errorf("got (%q, %v), want (my-secret, Secret)", tok, method)
		}
	})

	t.Run("nothing -> None", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		tok, method := GetAuth(cfg, r)
		if tok != "" || method != MethodNone {
			t.Errorf("got (%q, %v), want (\"\", None)", tok, method)
		}
	})

	t.Run("cookie takes precedence over header", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{Name: cookieName(cfg), Value: "cookie-session"})
		r.Header.Set("Authorization", "Bearer header-secret")
		tok, method := GetAuth(cfg, r)
		if tok != "cookie-session" || method != MethodSSO {
			t.Errorf("got (%q, %v), want cookie to win", tok, method)
		}
	})

	t.Run("legacy cookie -> SSO", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{Name: legacyCookieName(cfg), Value: "legacy-token"})
		tok, method := GetAuth(cfg, r)
		if tok != "legacy-token" || method != MethodSSO {
			t.Errorf("got (%q, %v), want (legacy-token, SSO)", tok, method)
		}
	})

	t.Run("current cookie takes precedence over legacy", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{Name: cookieName(cfg), Value: "current-token"})
		r.AddCookie(&http.Cookie{Name: legacyCookieName(cfg), Value: "legacy-token"})
		tok, method := GetAuth(cfg, r)
		if tok != "current-token" || method != MethodSSO {
			t.Errorf("got (%q, %v), want current cookie to win", tok, method)
		}
	})
}

// A session presented under the legacy cookie name gets re-issued under the
// current name and the legacy cookie is expired.
func TestMigrateLegacyCookie(t *testing.T) {
	cfg := testCfg()

	t.Run("legacy only -> re-issued and legacy expired", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{Name: legacyCookieName(cfg), Value: "the-session-id"})
		w := httptest.NewRecorder()
		MigrateLegacyCookie(cfg, w, r)

		var issued, expired *http.Cookie
		for _, c := range w.Result().Cookies() {
			switch c.Name {
			case cookieName(cfg):
				issued = c
			case legacyCookieName(cfg):
				expired = c
			}
		}
		if issued == nil {
			t.Fatal("expected session to be re-issued under the current cookie name")
		}
		if issued.Value != "the-session-id" {
			t.Errorf("re-issued cookie value = %q, want the same session id", issued.Value)
		}
		if expired == nil {
			t.Fatal("expected legacy cookie to be expired")
		}
		if expired.Value != "" || !expired.Expires.Before(time.Now()) {
			t.Errorf("legacy cookie = %+v, want empty and expired", expired)
		}
	})

	t.Run("current cookie present -> no-op", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(&http.Cookie{Name: cookieName(cfg), Value: "current-session"})
		r.AddCookie(&http.Cookie{Name: legacyCookieName(cfg), Value: "legacy-session"})
		w := httptest.NewRecorder()
		MigrateLegacyCookie(cfg, w, r)
		if n := len(w.Result().Cookies()); n != 0 {
			t.Errorf("expected no cookies set, got %d", n)
		}
	})

	t.Run("no legacy cookie -> no-op", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		MigrateLegacyCookie(cfg, w, r)
		if n := len(w.Result().Cookies()); n != 0 {
			t.Errorf("expected no cookies set, got %d", n)
		}
	})
}

func TestMethodString(t *testing.T) {
	cases := map[Method]string{
		MethodNone:     "none",
		MethodSSO:      "sso",
		MethodSecret:   "secret",
		MethodExcluded: "excluded",
		MethodUnknown:  "unknown",
		Method(99):     "unknown",
	}
	for m, want := range cases {
		if got := m.String(); got != want {
			t.Errorf("Method(%d).String() = %q, want %q", m, got, want)
		}
	}
}

// ClearCookie expires the session cookie under both the current and legacy
// names, so a legacy-cookie session can't survive a logout.
func TestClearCookie(t *testing.T) {
	cfg := testCfg()
	w := httptest.NewRecorder()
	ClearCookie(cfg, w)

	cookies := w.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}
	wantNames := map[string]bool{cookieName(cfg): false, legacyCookieName(cfg): false}
	for _, c := range cookies {
		if _, ok := wantNames[c.Name]; !ok {
			t.Errorf("unexpected cookie %q cleared", c.Name)
			continue
		}
		wantNames[c.Name] = true
		if c.Value != "" {
			t.Errorf("cleared cookie %q value = %q, want empty", c.Name, c.Value)
		}
		if !c.Expires.Before(time.Now()) {
			t.Errorf("cleared cookie %q expiry = %v, want in the past", c.Name, c.Expires)
		}
	}
	for name, seen := range wantNames {
		if !seen {
			t.Errorf("cookie %q was not cleared", name)
		}
	}
}

// GetUserID dispatches by auth method: SSO resolves the session, Secret looks
// up the DB, everything else returns 0.
func TestGetUserIDDispatch(t *testing.T) {
	ctx := context.Background()

	t.Run("SSO resolves session", func(t *testing.T) {
		a := &Auth{store: newFakeStore()}
		id := seedSession(t, a, session{
			UserID:      11,
			Sub:         "github|11",
			TokenExpiry: time.Now().Add(time.Hour),
			CreatedAt:   time.Now(),
		})
		uid, err := a.GetUserID(ctx, id, MethodSSO)
		if err != nil || uid != 11 {
			t.Errorf("SSO GetUserID = (%d, %v), want (11, nil)", uid, err)
		}
	})

	t.Run("Secret consults db", func(t *testing.T) {
		var lastSecret string
		db := dbtest.New()
		db.GetUserBySecretFn = func(s string) (int, error) {
			lastSecret = s
			return 5, nil
		}
		a := &Auth{db: db}
		uid, err := a.GetUserID(ctx, "some-secret", MethodSecret)
		if err != nil || uid != 5 {
			t.Errorf("Secret GetUserID = (%d, %v), want (5, nil)", uid, err)
		}
		if lastSecret != "some-secret" {
			t.Errorf("db queried with %q, want some-secret", lastSecret)
		}
	})

	t.Run("Secret db error propagates", func(t *testing.T) {
		db := dbtest.New()
		db.GetUserBySecretFn = func(string) (int, error) { return 0, fmt.Errorf("no such secret") }
		a := &Auth{db: db}
		if _, err := a.GetUserID(ctx, "x", MethodSecret); err == nil {
			t.Error("expected error from secret lookup")
		}
	})

	t.Run("None returns zero", func(t *testing.T) {
		a := &Auth{}
		uid, err := a.GetUserID(ctx, "", MethodNone)
		if err != nil || uid != 0 {
			t.Errorf("None GetUserID = (%d, %v), want (0, nil)", uid, err)
		}
	})
}
