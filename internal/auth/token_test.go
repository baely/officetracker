package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database/dbtest"
)

func testCfg() config.IntegratedApp {
	return config.IntegratedApp{
		SigningKey: "test-signing-key",
		App:        config.App{Env: "cloud"},
		Domain:     config.Domain{Domain: "officetracker.com.au"},
	}
}

// signClaims signs an arbitrary claim set with the config's key so tests can
// craft tokens that exercise each validation branch.
func signClaims(t *testing.T, cfg config.IntegratedApp, claims tokenClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString(signingKey(cfg))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
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

// A freshly generated token validates back to the same user id.
func TestTokenRoundTrip(t *testing.T) {
	cfg := testCfg()
	token, err := generateToken(cfg, 7)
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	uid, err := getUserIDFromToken(cfg, token)
	if err != nil {
		t.Fatalf("getUserIDFromToken: %v", err)
	}
	if uid != 7 {
		t.Errorf("uid = %d, want 7", uid)
	}
}

func TestTokenTampered(t *testing.T) {
	cfg := testCfg()
	token, _ := generateToken(cfg, 7)
	// Flip the final character of the signature.
	tampered := token[:len(token)-1]
	if token[len(token)-1] == 'a' {
		tampered += "b"
	} else {
		tampered += "a"
	}
	if _, err := getUserIDFromToken(cfg, tampered); err == nil {
		t.Fatal("tampered token should not validate")
	}
}

// A token signed with a different key must be rejected (alg confusion / forged
// signature protection).
func TestTokenWrongKey(t *testing.T) {
	signer := testCfg()
	token, _ := generateToken(signer, 7)

	verifier := testCfg()
	verifier.SigningKey = "a-completely-different-key"
	if _, err := getUserIDFromToken(verifier, token); err == nil {
		t.Fatal("token signed with a different key should not validate")
	}
}

func TestTokenExpired(t *testing.T) {
	cfg := testCfg()
	orig := loginExpiration
	loginExpiration = -time.Hour // issue an already-expired token
	token, _ := generateToken(cfg, 7)
	loginExpiration = orig

	_, err := getUserIDFromToken(cfg, token)
	if err == nil || err.Error() != "token expired" {
		t.Fatalf("expired token err = %v, want \"token expired\"", err)
	}
}

// Each missing/mismatched-claim branch returns its distinct error.
func TestTokenClaimValidation(t *testing.T) {
	cfg := testCfg()
	now := time.Now()
	iss := "officetracker.com.au"

	cases := []struct {
		name    string
		claims  tokenClaims
		wantErr string
	}{
		{
			name: "missing iat",
			claims: tokenClaims{RegisteredClaims: jwt.RegisteredClaims{
				Subject: "7", Issuer: iss, ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			}, User: 7},
			wantErr: "token missing required iat claim",
		},
		{
			name: "missing exp",
			claims: tokenClaims{RegisteredClaims: jwt.RegisteredClaims{
				Subject: "7", Issuer: iss, IssuedAt: jwt.NewNumericDate(now),
			}, User: 7},
			wantErr: "token missing required exp claim",
		},
		{
			name: "missing iss",
			claims: tokenClaims{RegisteredClaims: jwt.RegisteredClaims{
				Subject: "7", IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			}, User: 7},
			wantErr: "token missing required iss claim",
		},
		{
			name: "wrong iss",
			claims: tokenClaims{RegisteredClaims: jwt.RegisteredClaims{
				Subject: "7", Issuer: "evil.example.com", IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			}, User: 7},
			wantErr: "invalid token issuer",
		},
		{
			name: "missing sub",
			claims: tokenClaims{RegisteredClaims: jwt.RegisteredClaims{
				Issuer: iss, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			}, User: 7},
			wantErr: "token missing required sub claim",
		},
		{
			name: "subject mismatch",
			claims: tokenClaims{RegisteredClaims: jwt.RegisteredClaims{
				Subject: "999", Issuer: iss, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			}, User: 7},
			wantErr: "token subject mismatch",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			token := signClaims(t, cfg, c.claims)
			_, err := getUserIDFromToken(cfg, token)
			if err == nil || err.Error() != c.wantErr {
				t.Fatalf("err = %v, want %q", err, c.wantErr)
			}
		})
	}
}

// The issuer is derived from the qualified domain, so a config with a subdomain
// still round-trips.
func TestTokenIssuerWithSubdomain(t *testing.T) {
	cfg := testCfg()
	cfg.Domain = config.Domain{Subdomain: "app", Domain: "officetracker.com.au"}
	token, _ := generateToken(cfg, 3)
	if _, err := getUserIDFromToken(cfg, token); err != nil {
		t.Fatalf("subdomain issuer round-trip failed: %v", err)
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
		r.AddCookie(&http.Cookie{Name: cookieName(cfg), Value: "the-token"})
		tok, method := GetAuth(cfg, r)
		if tok != "the-token" || method != MethodSSO {
			t.Errorf("got (%q, %v), want (the-token, SSO)", tok, method)
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
		r.AddCookie(&http.Cookie{Name: cookieName(cfg), Value: "cookie-token"})
		r.Header.Set("Authorization", "Bearer header-secret")
		tok, method := GetAuth(cfg, r)
		if tok != "cookie-token" || method != MethodSSO {
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
		r.AddCookie(&http.Cookie{Name: legacyCookieName(cfg), Value: "legacy-token"})
		w := httptest.NewRecorder()
		MigrateLegacyCookie(cfg, w, r, 13)

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
		uid, err := getUserIDFromToken(cfg, issued.Value)
		if err != nil || uid != 13 {
			t.Errorf("re-issued cookie token validates to (%d, %v), want (13, nil)", uid, err)
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
		r.AddCookie(&http.Cookie{Name: cookieName(cfg), Value: "current-token"})
		r.AddCookie(&http.Cookie{Name: legacyCookieName(cfg), Value: "legacy-token"})
		w := httptest.NewRecorder()
		MigrateLegacyCookie(cfg, w, r, 13)
		if n := len(w.Result().Cookies()); n != 0 {
			t.Errorf("expected no cookies set, got %d", n)
		}
	})

	t.Run("no legacy cookie -> no-op", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		MigrateLegacyCookie(cfg, w, r, 13)
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

// issueToken mints a JWT and sets it as the session cookie; the cookie's token
// must validate back to the same user.
func TestIssueToken(t *testing.T) {
	cfg := testCfg()
	w := httptest.NewRecorder()
	if err := issueToken(cfg, w, 13); err != nil {
		t.Fatalf("issueToken: %v", err)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	c := cookies[0]
	if c.Name != cookieName(cfg) || !c.HttpOnly {
		t.Errorf("cookie = %+v, want httpOnly session cookie", c)
	}
	uid, err := getUserIDFromToken(cfg, c.Value)
	if err != nil || uid != 13 {
		t.Errorf("issued cookie token validates to (%d, %v), want (13, nil)", uid, err)
	}
}

// On localhost the cookie Domain is left blank so browsers accept it.
func TestIssueTokenLocalhostDomain(t *testing.T) {
	cfg := testCfg()
	cfg.Domain = config.Domain{Domain: "localhost"}
	w := httptest.NewRecorder()
	if err := issueToken(cfg, w, 1); err != nil {
		t.Fatalf("issueToken: %v", err)
	}
	if got := w.Result().Cookies()[0].Domain; got != "" {
		t.Errorf("localhost cookie domain = %q, want empty", got)
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

// GetUserID dispatches by auth method: SSO validates the token, Secret looks up
// the DB, everything else returns 0.
func TestGetUserIDDispatch(t *testing.T) {
	cfg := testCfg()
	token, _ := generateToken(cfg, 11)

	t.Run("SSO validates token", func(t *testing.T) {
		uid, err := GetUserID(cfg, nil, token, MethodSSO)
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
		uid, err := GetUserID(cfg, db, "some-secret", MethodSecret)
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
		if _, err := GetUserID(cfg, db, "x", MethodSecret); err == nil {
			t.Error("expected error from secret lookup")
		}
	})

	t.Run("None returns zero", func(t *testing.T) {
		uid, err := GetUserID(cfg, nil, "", MethodNone)
		if err != nil || uid != 0 {
			t.Errorf("None GetUserID = (%d, %v), want (0, nil)", uid, err)
		}
	})
}
