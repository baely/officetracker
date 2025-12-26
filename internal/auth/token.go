package auth

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/util"
)

const (
	userCookieBase = "user"
	debugCookie    = "debug"
	demoUserId     = "42069"
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

var (
	loginExpiration = time.Hour * 24 * 30
)

type tokenClaims struct {
	jwt.RegisteredClaims
	User int `json:"user"`
}

func signingKey(cfg config.IntegratedApp) []byte {
	return []byte(cfg.SigningKey)
}

func getValidationOptions() jwt.ParserOption {
	return jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name})
}

func GetUserID(cfg config.AppConfigurer, db database.Databaser, token string, authMethod Method) (int, error) {
	switch cfg := cfg.(type) {
	case config.IntegratedApp:
		if cfg.App.Demo {
			return 1, nil
		}
	}

	switch authMethod {
	case MethodSSO:
		return getUserIDFromToken(cfg.(config.IntegratedApp), token)
	case MethodSecret:
		return getUserIDFromSecret(db, token)
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

func generateToken(cfg config.IntegratedApp, userID int) (string, error) {
	now := time.Now()
	claims := tokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			Issuer:    util.QualifiedDomain(cfg.Domain),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(loginExpiration)),
		},
		User: userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(signingKey(cfg))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func issueToken(cfg config.IntegratedApp, w http.ResponseWriter, userID int) error {
	token, err := generateToken(cfg, userID)
	if err != nil {
		return err
	}

	domain := util.QualifiedDomain(cfg.Domain)
	if domain == "localhost" {
		domain = ""
	}

	cookie := http.Cookie{
		Name:     cookieName(cfg),
		Value:    token,
		Path:     util.BasePath(cfg.Domain),
		Expires:  time.Now().Add(loginExpiration),
		Domain:   util.QualifiedDomain(cfg.Domain),
		HttpOnly: true,
		Secure:   false,
	}
	//slog.Info(fmt.Sprintf("Issuing cookie for user %d", userID))
	slog.Info("minted new jwt",
		"userID", userID,
		"expiresAt", time.Now().Add(loginExpiration).Format(time.RFC3339))
	http.SetCookie(w, &cookie)

	return nil
}

func getUserIDFromToken(cfg config.IntegratedApp, token string) (int, error) {
	claims := &tokenClaims{}

	t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return signingKey(cfg), nil
	}, getValidationOptions())

	if err != nil {
		// Check if error is specifically due to token expiration
		if errors.Is(err, jwt.ErrTokenExpired) {
			slog.Info("token validation failed: token expired", "userID", claims.User)
			return 0, fmt.Errorf("token expired")
		}

		// For other parsing errors, log and return
		slog.Warn("token validation failed", "error", err.Error())
		return 0, err
	}

	if !t.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	// Validate required claims - all must be present (no backward compatibility)
	if claims.IssuedAt == nil {
		slog.Warn("token validation failed: missing iat claim")
		return 0, fmt.Errorf("token missing required iat claim")
	}

	if claims.ExpiresAt == nil {
		slog.Warn("token validation failed: missing exp claim")
		return 0, fmt.Errorf("token missing required exp claim")
	}

	// Validate issuer (required)
	expectedIssuer := util.QualifiedDomain(cfg.Domain)
	if claims.Issuer == "" {
		slog.Warn("token validation failed: missing iss claim")
		return 0, fmt.Errorf("token missing required iss claim")
	}
	if claims.Issuer != expectedIssuer {
		slog.Warn("token validation failed: invalid issuer",
			"expected", expectedIssuer,
			"actual", claims.Issuer)
		return 0, fmt.Errorf("invalid token issuer")
	}

	// Validate subject (required) and ensure it matches user ID
	expectedSubject := fmt.Sprintf("%d", claims.User)
	if claims.Subject == "" {
		slog.Warn("token validation failed: missing sub claim")
		return 0, fmt.Errorf("token missing required sub claim")
	}
	if claims.Subject != expectedSubject {
		slog.Warn("token validation failed: subject/user mismatch",
			"subject", claims.Subject,
			"user", claims.User)
		return 0, fmt.Errorf("token subject mismatch")
	}

	return claims.User, nil
}

func validateDevSecret(secret string) string {
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
