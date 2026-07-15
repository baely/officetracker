package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/util"
)

// Login sessions are managed by Auth0: the cookie holds only an opaque session
// ID, and the session record in Redis keeps the Auth0-issued token set. A
// session stays alive by refreshing with Auth0 whenever its token expires, so
// revoking the user's grant at Auth0 ends the session at the next refresh, and
// logout revokes the refresh token so it can't mint new tokens.

const (
	// sessionLifetime caps a session's absolute age. It matches Auth0's
	// default refresh token absolute lifetime (30 days), beyond which Auth0
	// would refuse to refresh anyway.
	sessionLifetime = 30 * 24 * time.Hour

	// tokenExpiryLeeway refreshes slightly early so a token never lapses
	// mid-request.
	tokenExpiryLeeway = 30 * time.Second

	// refreshLockTTL bounds how long a crashed refresh holds the lock.
	refreshLockTTL = 10 * time.Second

	// fallbackTokenLifetime is used when Auth0 doesn't report a token expiry.
	fallbackTokenLifetime = time.Hour

	// auth0Timeout bounds calls to Auth0 so a slow tenant can't stall
	// request handling.
	auth0Timeout = 10 * time.Second

	// refreshGraceWindow keeps sessions alive on stale tokens while Auth0
	// is unreachable. Only definitive rejections (Auth0 answering that the
	// grant is gone) end a session inside this window; transient failures
	// serve the stale session and retry. A session whose token has been
	// expired longer than this is ended regardless.
	refreshGraceWindow = 24 * time.Hour

	// refreshRetryInterval spaces out refresh attempts during an outage so
	// at most one request per session eats the Auth0 timeout per interval.
	refreshRetryInterval = time.Minute
)

var errSessionInvalid = errors.New("session invalid")

type session struct {
	UserID       int       `json:"user_id"`
	Sub          string    `json:"sub"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenExpiry  time.Time `json:"token_expiry"`
	CreatedAt    time.Time `json:"created_at"`
	// RefreshRetryAt is set after a transient refresh failure; until then
	// requests serve the stale session without contacting Auth0 again.
	RefreshRetryAt time.Time `json:"refresh_retry_at,omitempty"`
}

func sessionKey(id string) string {
	return "auth0:session:" + id
}

func refreshLockKey(id string) string {
	return "auth0:session:refresh:" + id
}

func newSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate session id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func tokenExpiry(token *oauth2.Token) time.Time {
	if token.Expiry.IsZero() {
		return time.Now().Add(fallbackTokenLifetime)
	}
	return token.Expiry
}

// CreateSession stores the Auth0 token set against a new opaque session ID and
// sets that ID as the session cookie.
func (a *Auth) CreateSession(ctx context.Context, cfg config.IntegratedApp, w http.ResponseWriter, userID int, sub string, token *oauth2.Token) error {
	id, err := newSessionID()
	if err != nil {
		return err
	}

	if token.RefreshToken == "" {
		slog.Warn("auth0 returned no refresh token; session will end when the token expires — enable the Refresh Token grant on the Auth0 application",
			"userID", userID)
	}

	sess := session{
		UserID:       userID,
		Sub:          sub,
		RefreshToken: token.RefreshToken,
		TokenExpiry:  tokenExpiry(token),
		CreatedAt:    time.Now(),
	}
	if err := a.saveSession(ctx, id, sess); err != nil {
		return err
	}

	issueSessionCookie(cfg, w, id)
	slog.Info("created auth0-backed session",
		"userID", userID,
		"tokenExpiresAt", sess.TokenExpiry.Format(time.RFC3339))
	return nil
}

// saveSession persists the session with a TTL that preserves its absolute
// lifetime across refreshes.
func (a *Auth) saveSession(ctx context.Context, id string, sess session) error {
	remaining := sessionLifetime - time.Since(sess.CreatedAt)
	if remaining <= 0 {
		return errSessionInvalid
	}
	b, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	if err := a.store.SetState(ctx, sessionKey(id), string(b), remaining); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}
	return nil
}

func (a *Auth) getSession(ctx context.Context, id string) (session, error) {
	raw, err := a.store.GetState(ctx, sessionKey(id))
	if err != nil {
		return session{}, errSessionInvalid
	}
	var sess session
	if err := json.Unmarshal([]byte(raw), &sess); err != nil {
		return session{}, errSessionInvalid
	}
	return sess, nil
}

func (a *Auth) deleteSession(ctx context.Context, id string) {
	if err := a.store.DeleteState(ctx, sessionKey(id)); err != nil {
		slog.Warn("failed to delete session", "error", err.Error())
	}
}

// userIDFromSession resolves a session cookie value to a user, refreshing the
// Auth0 token set when it has expired. A session Auth0 will no longer refresh
// is deleted and reported invalid.
func (a *Auth) userIDFromSession(ctx context.Context, id string) (int, error) {
	sess, err := a.getSession(ctx, id)
	if err != nil {
		return 0, err
	}

	if time.Until(sess.TokenExpiry) > tokenExpiryLeeway {
		return sess.UserID, nil
	}

	return a.refreshSession(ctx, id, sess)
}

// refreshSession exchanges the stored refresh token for a fresh Auth0 token
// set. Auth0 rotates refresh tokens, so concurrent requests coordinate through
// a short-lived lock: the loser waits for the winner's result rather than
// burning the already-rotated token.
func (a *Auth) refreshSession(ctx context.Context, id string, sess session) (int, error) {
	if sess.RefreshToken == "" {
		slog.Info("session token expired with no refresh token held; ending session", "userID", sess.UserID)
		a.deleteSession(ctx, id)
		return 0, errSessionInvalid
	}

	// A recent attempt failed transiently; serve the stale session and wait
	// out the retry interval instead of hammering Auth0.
	if withinRefreshGrace(sess) && time.Now().Before(sess.RefreshRetryAt) {
		return sess.UserID, nil
	}

	locked, err := a.store.SetStateNX(ctx, refreshLockKey(id), 1, refreshLockTTL)
	if err != nil {
		return 0, fmt.Errorf("failed to acquire refresh lock: %w", err)
	}
	if !locked {
		return a.awaitRefresh(ctx, id, sess)
	}
	defer func() {
		if err := a.store.DeleteState(ctx, refreshLockKey(id)); err != nil {
			slog.Warn("failed to release refresh lock", "error", err.Error())
		}
	}()

	refreshCtx, cancel := context.WithTimeout(ctx, auth0Timeout)
	defer cancel()
	token, err := a.Auth0OauthCfg().TokenSource(refreshCtx, &oauth2.Token{RefreshToken: sess.RefreshToken}).Token()
	if err != nil {
		if refreshRejected(err) {
			slog.Info("auth0 refused token refresh; ending session", "userID", sess.UserID, "error", err.Error())
			a.deleteSession(ctx, id)
			return 0, errSessionInvalid
		}
		// Auth0 is unreachable or erroring, not rejecting the grant.
		if withinRefreshGrace(sess) {
			slog.Warn("auth0 unreachable for token refresh; serving stale session within grace window",
				"userID", sess.UserID, "error", err.Error())
			sess.RefreshRetryAt = time.Now().Add(refreshRetryInterval)
			if err := a.saveSession(ctx, id, sess); err != nil {
				slog.Warn("failed to record refresh retry time", "error", err.Error())
			}
			return sess.UserID, nil
		}
		slog.Warn("auth0 unreachable and refresh grace window exhausted; ending session",
			"userID", sess.UserID, "error", err.Error())
		a.deleteSession(ctx, id)
		return 0, errSessionInvalid
	}

	// The refresh response carries a new ID token; verify it and check it
	// still identifies the same subject before trusting the new token set.
	if rawIDToken, ok := token.Extra("id_token").(string); ok {
		idToken, err := a.provider.Verifier(&oidc.Config{ClientID: a.auth0Cfg.ClientID}).Verify(ctx, rawIDToken)
		if err != nil {
			slog.Error("refreshed id token failed verification; ending session", "userID", sess.UserID, "error", err.Error())
			a.deleteSession(ctx, id)
			return 0, errSessionInvalid
		}
		if idToken.Subject != sess.Sub {
			slog.Error("refreshed id token subject mismatch; ending session", "userID", sess.UserID)
			a.deleteSession(ctx, id)
			return 0, errSessionInvalid
		}
	}

	if token.RefreshToken != "" {
		sess.RefreshToken = token.RefreshToken // rotated by Auth0
	}
	sess.TokenExpiry = tokenExpiry(token)
	sess.RefreshRetryAt = time.Time{}
	if err := a.saveSession(ctx, id, sess); err != nil {
		return 0, err
	}

	slog.Info("refreshed auth0 session",
		"userID", sess.UserID,
		"tokenExpiresAt", sess.TokenExpiry.Format(time.RFC3339))
	return sess.UserID, nil
}

// awaitRefresh polls for the outcome of a refresh underway on another request.
func (a *Auth) awaitRefresh(ctx context.Context, id string, stale session) (int, error) {
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
		sess, err := a.getSession(ctx, id)
		if err != nil {
			// The concurrent refresh failed and deleted the session.
			return 0, err
		}
		if time.Until(sess.TokenExpiry) > tokenExpiryLeeway {
			return sess.UserID, nil
		}
		if withinRefreshGrace(sess) && time.Now().Before(sess.RefreshRetryAt) {
			// The concurrent refresh hit an Auth0 outage; serve stale.
			return sess.UserID, nil
		}
	}
	// The concurrent refresh hasn't landed; serve this request on the stale
	// session rather than logging the user out spuriously.
	slog.Warn("session refresh still in flight; allowing request on stale session", "userID", stale.UserID)
	return stale.UserID, nil
}

// withinRefreshGrace reports whether the session's token expired recently
// enough that a transient Auth0 failure shouldn't end the session.
func withinRefreshGrace(sess session) bool {
	return time.Since(sess.TokenExpiry) < refreshGraceWindow
}

// refreshRejected reports whether a refresh error is Auth0 definitively
// rejecting the grant (revoked or expired), as opposed to Auth0 being
// unreachable or erroring.
func refreshRejected(err error) bool {
	var re *oauth2.RetrieveError
	if errors.As(err, &re) && re.Response != nil {
		return re.Response.StatusCode >= 400 && re.Response.StatusCode < 500
	}
	return false
}

// MigrateLegacyCookie re-issues a session presented under the legacy cookie
// name using the current cookie name, and expires the legacy cookie. No-op
// unless the request authenticated via the legacy cookie alone.
func MigrateLegacyCookie(cfg config.IntegratedApp, w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie(cookieName(cfg)); err == nil {
		return
	}
	legacy, err := r.Cookie(legacyCookieName(cfg))
	if err != nil || legacy.Value == "" {
		return
	}
	issueSessionCookie(cfg, w, legacy.Value)
	expireCookie(cfg, w, legacyCookieName(cfg))
}

// sessionIDFromRequest returns the session ID presented on the request,
// checking the current cookie name before the legacy one.
func sessionIDFromRequest(cfg config.IntegratedApp, r *http.Request) string {
	if cookie, err := r.Cookie(cookieName(cfg)); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	if cookie, err := r.Cookie(legacyCookieName(cfg)); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return ""
}

// Logout ends the session both locally and at Auth0: the refresh token is
// revoked so it can no longer mint tokens, and the session record and cookie
// are removed.
func (a *Auth) Logout(ctx context.Context, cfg config.IntegratedApp, w http.ResponseWriter, r *http.Request) {
	defer ClearCookie(cfg, w)

	id := sessionIDFromRequest(cfg, r)
	if id == "" {
		return
	}

	sess, err := a.getSession(ctx, id)
	if err == nil && sess.RefreshToken != "" {
		if err := a.revokeRefreshToken(ctx, sess.RefreshToken); err != nil {
			slog.Warn("failed to revoke refresh token at auth0", "userID", sess.UserID, "error", err.Error())
		} else {
			slog.Info("revoked refresh token at auth0", "userID", sess.UserID)
		}
	}
	a.deleteSession(ctx, id)
}

// revokeRefreshToken invalidates a refresh token at Auth0.
// https://auth0.com/docs/api/authentication#revoke-refresh-token
func (a *Auth) revokeRefreshToken(ctx context.Context, refreshToken string) error {
	ctx, cancel := context.WithTimeout(ctx, auth0Timeout)
	defer cancel()

	form := url.Values{
		"client_id":     {a.auth0Cfg.ClientID},
		"client_secret": {a.auth0Cfg.ClientSecret},
		"token":         {refreshToken},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.revocationEndpoint(), strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("revocation endpoint returned %s", resp.Status)
	}
	return nil
}

func (a *Auth) revocationEndpoint() string {
	var claims struct {
		RevocationEndpoint string `json:"revocation_endpoint"`
	}
	if err := a.provider.Claims(&claims); err == nil && claims.RevocationEndpoint != "" {
		return claims.RevocationEndpoint
	}
	// Auth0 serves revocation at /oauth/revoke alongside /oauth/token.
	return strings.TrimSuffix(a.provider.Endpoint().TokenURL, "/token") + "/revoke"
}

func issueSessionCookie(cfg config.IntegratedApp, w http.ResponseWriter, sessionID string) {
	domain := util.QualifiedDomain(cfg.Domain)
	if domain == "localhost" {
		domain = ""
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName(cfg),
		Value:    sessionID,
		Path:     util.BasePath(cfg.Domain),
		Expires:  time.Now().Add(sessionLifetime),
		Domain:   domain,
		HttpOnly: true,
		Secure:   secureCookies(cfg),
		SameSite: http.SameSiteLaxMode,
	})
}

func secureCookies(cfg config.IntegratedApp) bool {
	return cfg.Domain.Protocol == "https"
}
