package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
)

const (
	userCookieBase = "__session"
)

func cookieName(cfg config.IntegratedApp) string {
	if cfg.App.Env == "" || cfg.App.Env == "cloud" {
		return userCookieBase
	}
	return userCookieBase + "_" + cfg.App.Env
}

type Method int

const (
	MethodUnknown = Method(iota)
	MethodNone
	MethodSSO
	MethodSecret
	MethodExcluded
)

// GetUserID resolves the presented credential to a user ID. SSO credentials
// are opaque session IDs backed by Auth0-managed token sets; secrets are API
// tokens looked up in the database.
func (a *Auth) GetUserID(ctx context.Context, token string, authMethod Method) (int, error) {
	switch authMethod {
	case MethodSSO:
		return a.userIDFromSession(ctx, token)
	case MethodSecret:
		return getUserIDFromSecret(a.db, token)
	default:
		return 0, nil
	}
}

func getUserIDFromSecret(db database.Databaser, token string) (int, error) {
	userID, err := db.GetUserBySecret(token)
	if err != nil {
		err = fmt.Errorf("failed to get user id from secret: %w", err)
		slog.Error(err.Error())
		return 0, err
	}
	return userID, nil
}

func validateDevSecret(secret string) string {
	if secret == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(secret), "bearer ") {
		slog.Warn("invalid secret format")
		return ""
	}
	secret = secret[7:]
	return secret
}

func GetAuth(cfg config.IntegratedApp, r *http.Request) (string, Method) {
	// try to get from cookie
	cookie, err := r.Cookie(cookieName(cfg))
	if err == nil && cookie != nil {
		return cookie.Value, MethodSSO
	}

	// try to get from header
	secret := r.Header.Get("Authorization")
	secret = validateDevSecret(secret)
	if secret != "" {
		return secret, MethodSecret
	}

	return "", MethodNone
}

func (m Method) String() string {
	switch m {
	case MethodNone:
		return "none"
	case MethodSSO:
		return "sso"
	case MethodSecret:
		return "secret"
	case MethodExcluded:
		return "excluded"
	default:
		return "unknown"
	}
}
