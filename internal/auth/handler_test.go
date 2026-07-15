package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baely/officetracker/internal/config"
)

// handleLogout clears the session cookie and redirects home.
func TestHandleLogout(t *testing.T) {
	cfg := testCfg()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/logout", nil)

	handleLogout(cfg, &Auth{store: newFakeStore()})(w, r)

	res := w.Result()
	if res.StatusCode != http.StatusTemporaryRedirect {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusTemporaryRedirect)
	}
	if loc := res.Header.Get("Location"); loc != "/" {
		t.Errorf("redirect location = %q, want /", loc)
	}

	var cleared *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == cookieName(cfg) {
			cleared = c
		}
	}
	if cleared == nil {
		t.Fatal("logout did not set the session cookie")
	}
	if cleared.Value != "" {
		t.Errorf("logout cookie value = %q, want empty", cleared.Value)
	}
}

// The cookie name switches per environment, so logout in a dev environment
// clears the env-suffixed cookie.
func TestHandleLogoutDevEnv(t *testing.T) {
	cfg := config.IntegratedApp{
		App:    config.App{Env: "dev"},
		Domain: config.Domain{Domain: "localhost"},
	}
	w := httptest.NewRecorder()
	handleLogout(cfg, &Auth{store: newFakeStore()})(w, httptest.NewRequest("GET", "/logout", nil))

	found := false
	for _, c := range w.Result().Cookies() {
		if c.Name == "__session_dev" {
			found = true
		}
	}
	if !found {
		t.Error("expected __session_dev cookie to be cleared in dev env")
	}
}
