package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// handleLogout clears the session cookie and redirects home.
func TestHandleLogout(t *testing.T) {
	cfg := testCfg()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/logout", nil)

	handleLogout(cfg)(w, r)

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
