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
		{"", "__session"},
		{"cloud", "__session"},
		{"dev", "__session_dev"},
		{"staging", "__session_staging"},
	}
	for _, c := range cases {
		cfg := config.IntegratedApp{App: config.App{Env: c.env}}
		if got := cookieName(cfg); got != c.want {
			t.Errorf("cookieName(env=%q) = %q, want %q", c.env, got, c.want)
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

func TestClearCookie(t *testing.T) {
	cfg := testCfg()
	w := httptest.NewRecorder()
	ClearCookie(cfg, w)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	c := cookies[0]
	if c.Name != cookieName(cfg) {
		t.Errorf("cookie name = %q, want %q", c.Name, cookieName(cfg))
	}
	if c.Value != "" {
		t.Errorf("cleared cookie value = %q, want empty", c.Value)
	}
	if !c.Expires.Before(time.Now()) {
		t.Errorf("cleared cookie expiry = %v, want in the past", c.Expires)
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
