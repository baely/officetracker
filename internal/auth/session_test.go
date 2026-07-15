package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/baely/officetracker/internal/config"
)

// fakeStore is an in-memory SessionStore. TTLs are accepted but not enforced;
// SetStateNX honours key existence so lock behaviour is real.
type fakeStore struct {
	mu   sync.Mutex
	data map[string]string
}

func newFakeStore() *fakeStore {
	return &fakeStore{data: map[string]string{}}
}

func (f *fakeStore) SetState(_ context.Context, key string, value interface{}, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[key] = fmt.Sprintf("%v", value)
	return nil
}

func (f *fakeStore) GetState(_ context.Context, key string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.data[key]
	if !ok {
		return "", errors.New("no such key")
	}
	return v, nil
}

func (f *fakeStore) GetStateInt(ctx context.Context, key string) (int, error) {
	v, err := f.GetState(ctx, key)
	if err != nil {
		return 0, err
	}
	var i int
	_, err = fmt.Sscanf(v, "%d", &i)
	return i, err
}

func (f *fakeStore) SetStateNX(_ context.Context, key string, value interface{}, _ time.Duration) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.data[key]; ok {
		return false, nil
	}
	f.data[key] = fmt.Sprintf("%v", value)
	return true, nil
}

func (f *fakeStore) DeleteState(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.data, key)
	return nil
}

func (f *fakeStore) has(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.data[key]
	return ok
}

// seedSession stores a session directly and returns its ID.
func seedSession(t *testing.T, a *Auth, sess session) string {
	t.Helper()
	id, err := newSessionID()
	if err != nil {
		t.Fatalf("newSessionID: %v", err)
	}
	if err := a.saveSession(context.Background(), id, sess); err != nil {
		t.Fatalf("saveSession: %v", err)
	}
	return id
}

// fakeAuth0 stands in for an Auth0 tenant: OIDC discovery plus configurable
// token and revocation endpoints.
type fakeAuth0 struct {
	srv *httptest.Server

	mu           sync.Mutex
	tokenForm    url.Values // last refresh request
	revokeForm   url.Values // last revocation request
	tokenHandler http.HandlerFunc
}

func newFakeAuth0(t *testing.T) *fakeAuth0 {
	t.Helper()
	f := &fakeAuth0{}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 f.srv.URL,
			"authorization_endpoint": f.srv.URL + "/authorize",
			"token_endpoint":         f.srv.URL + "/oauth/token",
			"jwks_uri":               f.srv.URL + "/jwks",
			"revocation_endpoint":    f.srv.URL + "/oauth/revoke",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"keys":[]}`))
	})
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		f.mu.Lock()
		f.tokenForm = r.PostForm
		handler := f.tokenHandler
		f.mu.Unlock()
		if handler != nil {
			handler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "access-2",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"refresh_token": "rt-2",
		})
	})
	mux.HandleFunc("/oauth/revoke", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		f.mu.Lock()
		f.revokeForm = r.PostForm
		f.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

// auth constructs an Auth wired to the fake tenant with a fresh fake store.
func (f *fakeAuth0) auth(t *testing.T) (*Auth, *fakeStore) {
	t.Helper()
	provider, err := oidc.NewProvider(context.Background(), f.srv.URL)
	if err != nil {
		t.Fatalf("oidc.NewProvider: %v", err)
	}
	store := newFakeStore()
	return &Auth{
		baseUri:  "https://officetracker.com.au/",
		auth0Cfg: &config.Auth0{Domain: f.srv.URL, ClientID: "cid", ClientSecret: "csecret"},
		store:    store,
		provider: provider,
	}, store
}

// A freshly created session sets an opaque HttpOnly cookie whose value
// resolves straight back to the user without contacting Auth0.
func TestCreateSessionRoundTrip(t *testing.T) {
	cfg := testCfg()
	a := &Auth{store: newFakeStore()}
	w := httptest.NewRecorder()

	token := &oauth2.Token{RefreshToken: "rt-1", Expiry: time.Now().Add(time.Hour)}
	if err := a.CreateSession(context.Background(), cfg, w, 7, "github|7", token); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	c := cookies[0]
	if c.Name != cookieName(cfg) || !c.HttpOnly {
		t.Errorf("cookie = %+v, want httpOnly session cookie", c)
	}
	if c.Value == "" || strings.Contains(c.Value, ".") {
		t.Errorf("cookie value = %q, want an opaque session id (not a JWT)", c.Value)
	}

	uid, err := a.userIDFromSession(context.Background(), c.Value)
	if err != nil || uid != 7 {
		t.Errorf("userIDFromSession = (%d, %v), want (7, nil)", uid, err)
	}
}

// On localhost the cookie Domain is left blank so browsers accept it.
func TestSessionCookieLocalhostDomain(t *testing.T) {
	cfg := testCfg()
	cfg.Domain = config.Domain{Domain: "localhost"}
	w := httptest.NewRecorder()
	issueSessionCookie(cfg, w, "some-id")
	if got := w.Result().Cookies()[0].Domain; got != "" {
		t.Errorf("localhost cookie domain = %q, want empty", got)
	}
}

// The Secure flag follows the configured protocol.
func TestSessionCookieSecureFlag(t *testing.T) {
	cases := []struct {
		protocol string
		want     bool
	}{
		{"https", true},
		{"http", false},
		{"", false},
	}
	for _, c := range cases {
		cfg := testCfg()
		cfg.Domain.Protocol = c.protocol
		w := httptest.NewRecorder()
		issueSessionCookie(cfg, w, "some-id")
		if got := w.Result().Cookies()[0].Secure; got != c.want {
			t.Errorf("protocol %q: Secure = %v, want %v", c.protocol, got, c.want)
		}
	}
}

func TestUnknownSessionRejected(t *testing.T) {
	a := &Auth{store: newFakeStore()}
	if _, err := a.userIDFromSession(context.Background(), "no-such-session"); err == nil {
		t.Fatal("unknown session id should not resolve")
	}
}

// An expired token with no refresh token can't be renewed: the session is
// deleted and reported invalid.
func TestExpiredSessionWithoutRefreshTokenEnds(t *testing.T) {
	a := &Auth{store: newFakeStore()}
	id := seedSession(t, a, session{
		UserID:      7,
		Sub:         "github|7",
		TokenExpiry: time.Now().Add(-time.Minute),
		CreatedAt:   time.Now(),
	})

	if _, err := a.userIDFromSession(context.Background(), id); err == nil {
		t.Fatal("expired session without refresh token should be invalid")
	}
	if a.store.(*fakeStore).has(sessionKey(id)) {
		t.Error("dead session should be deleted from the store")
	}
}

// An expired token with a refresh token is refreshed at Auth0; the rotated
// refresh token and new expiry are persisted.
func TestExpiredSessionRefreshesAndRotates(t *testing.T) {
	f := newFakeAuth0(t)
	a, store := f.auth(t)
	id := seedSession(t, a, session{
		UserID:       7,
		Sub:          "github|7",
		RefreshToken: "rt-1",
		TokenExpiry:  time.Now().Add(-time.Minute),
		CreatedAt:    time.Now(),
	})

	uid, err := a.userIDFromSession(context.Background(), id)
	if err != nil || uid != 7 {
		t.Fatalf("userIDFromSession = (%d, %v), want (7, nil)", uid, err)
	}

	f.mu.Lock()
	form := f.tokenForm
	f.mu.Unlock()
	if form.Get("grant_type") != "refresh_token" || form.Get("refresh_token") != "rt-1" {
		t.Errorf("refresh request form = %v, want refresh_token grant with rt-1", form)
	}

	sess, err := a.getSession(context.Background(), id)
	if err != nil {
		t.Fatalf("getSession after refresh: %v", err)
	}
	if sess.RefreshToken != "rt-2" {
		t.Errorf("stored refresh token = %q, want rotated rt-2", sess.RefreshToken)
	}
	if time.Until(sess.TokenExpiry) < 30*time.Minute {
		t.Errorf("stored token expiry = %v, want ~1h away", sess.TokenExpiry)
	}
	if store.has(refreshLockKey(id)) {
		t.Error("refresh lock should be released")
	}
}

// When Auth0 refuses the refresh (revoked / expired grant), the session is
// deleted: revocation at Auth0 ends the login.
func TestRefreshRefusedEndsSession(t *testing.T) {
	f := newFakeAuth0(t)
	f.tokenHandler = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"revoked"}`))
	}
	a, store := f.auth(t)
	id := seedSession(t, a, session{
		UserID:       7,
		Sub:          "github|7",
		RefreshToken: "rt-1",
		TokenExpiry:  time.Now().Add(-time.Minute),
		CreatedAt:    time.Now(),
	})

	if _, err := a.userIDFromSession(context.Background(), id); err == nil {
		t.Fatal("refused refresh should invalidate the session")
	}
	if store.has(sessionKey(id)) {
		t.Error("session should be deleted after a refused refresh")
	}
}

// While another request holds the refresh lock, this one waits and picks up
// the refreshed session instead of burning the rotated refresh token.
func TestConcurrentRefreshWaitsForWinner(t *testing.T) {
	a := &Auth{store: newFakeStore()}
	stale := session{
		UserID:       7,
		Sub:          "github|7",
		RefreshToken: "rt-1",
		TokenExpiry:  time.Now().Add(-time.Minute),
		CreatedAt:    time.Now(),
	}
	id := seedSession(t, a, stale)

	// Simulate a refresh in flight on another request.
	if _, err := a.store.SetStateNX(context.Background(), refreshLockKey(id), 1, refreshLockTTL); err != nil {
		t.Fatalf("seed lock: %v", err)
	}
	go func() {
		time.Sleep(300 * time.Millisecond)
		fresh := stale
		fresh.RefreshToken = "rt-2"
		fresh.TokenExpiry = time.Now().Add(time.Hour)
		_ = a.saveSession(context.Background(), id, fresh)
	}()

	start := time.Now()
	uid, err := a.userIDFromSession(context.Background(), id)
	if err != nil || uid != 7 {
		t.Fatalf("userIDFromSession = (%d, %v), want (7, nil)", uid, err)
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("waited %v for the concurrent refresh, want well under the full poll window", time.Since(start))
	}
}

// Logout revokes the refresh token at Auth0, deletes the session and clears
// the cookie.
func TestLogoutRevokesAtAuth0(t *testing.T) {
	f := newFakeAuth0(t)
	a, store := f.auth(t)
	cfg := testCfg()
	id := seedSession(t, a, session{
		UserID:       7,
		Sub:          "github|7",
		RefreshToken: "rt-1",
		TokenExpiry:  time.Now().Add(time.Hour),
		CreatedAt:    time.Now(),
	})

	r := httptest.NewRequest("GET", "/logout", nil)
	r.AddCookie(&http.Cookie{Name: cookieName(cfg), Value: id})
	w := httptest.NewRecorder()

	a.Logout(context.Background(), cfg, w, r)

	f.mu.Lock()
	form := f.revokeForm
	f.mu.Unlock()
	if form.Get("token") != "rt-1" || form.Get("client_id") != "cid" {
		t.Errorf("revocation form = %v, want token rt-1 for client cid", form)
	}
	if store.has(sessionKey(id)) {
		t.Error("session should be deleted on logout")
	}

	var cleared bool
	for _, c := range w.Result().Cookies() {
		if c.Name == cookieName(cfg) && c.Value == "" {
			cleared = true
		}
	}
	if !cleared {
		t.Error("logout should clear the session cookie")
	}
}

// Logout without a session cookie still clears the cookie and does nothing
// else.
func TestLogoutWithoutSession(t *testing.T) {
	a := &Auth{store: newFakeStore()}
	cfg := testCfg()
	w := httptest.NewRecorder()

	a.Logout(context.Background(), cfg, w, httptest.NewRequest("GET", "/logout", nil))

	// ClearCookie expires the session cookie under both the current and
	// legacy names.
	if len(w.Result().Cookies()) != 2 {
		t.Fatal("expected the clearing cookies to be set")
	}
}

// A session created without a refresh token still works until its token
// expires (Auth0 app without the refresh token grant).
func TestCreateSessionWithoutRefreshToken(t *testing.T) {
	cfg := testCfg()
	a := &Auth{store: newFakeStore()}
	w := httptest.NewRecorder()

	token := &oauth2.Token{Expiry: time.Now().Add(time.Hour)}
	if err := a.CreateSession(context.Background(), cfg, w, 7, "github|7", token); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	id := w.Result().Cookies()[0].Value
	if uid, err := a.userIDFromSession(context.Background(), id); err != nil || uid != 7 {
		t.Errorf("userIDFromSession = (%d, %v), want (7, nil)", uid, err)
	}
}
