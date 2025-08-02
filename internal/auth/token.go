package auth

import (
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
	userCookie = "user"
	demoUserId = "42069"
)

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
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": userID,
	})

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
		Name:     userCookie,
		Value:    token,
		Path:     util.BasePath(cfg.Domain),
		Expires:  time.Now().Add(loginExpiration),
		Domain:   util.QualifiedDomain(cfg.Domain),
		HttpOnly: true,
		Secure:   false,
	}
	//slog.Info(fmt.Sprintf("Issuing cookie for user %d", userID))
	slog.Info("minted new jwt", "userID", userID)
	http.SetCookie(w, &cookie)

	return nil
}

func getUserIDFromToken(cfg config.IntegratedApp, token string) (int, error) {
	claims := &tokenClaims{}

	t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return signingKey(cfg), nil
	})
	if err != nil {
		return 0, err
	}
	if !t.Valid {
		return 0, fmt.Errorf("invalid token")
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

func GetAuth(r *http.Request) (string, Method) {
	// try to get from cookie
	cookie, err := r.Cookie(userCookie)
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
