package auth

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/util"
)

const (
	userCookie = "user"
	demoUserId = "42069"
)

var (
	loginExpiration = time.Hour * 24 * 30
)

type tokenClaims struct {
	jwt.RegisteredClaims
	User string `json:"user"`
}

func signingKey(cfg config.IntegratedApp) []byte {
	return []byte(cfg.SigningKey)
}

func GetUserID(cfg config.IntegratedApp, r *http.Request) string {
	if cfg.App.Demo {
		return demoUserId
	}

	cookie, err := r.Cookie(userCookie)
	if err != nil {
		return ""
	}

	userID, err := getUserIDFromToken(cfg, cookie.Value)
	if err != nil {
		return ""
	}

	return userID
}

func generateToken(cfg config.IntegratedApp, userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": userID,
	})

	tokenString, err := token.SignedString(signingKey(cfg))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func issueToken(cfg config.IntegratedApp, w http.ResponseWriter, userID string) error {
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
	http.SetCookie(w, &cookie)

	slog.Info(fmt.Sprintf("issued token: %+v", cookie))

	return nil
}

func validateToken(cfg config.IntegratedApp, token string) error {
	claims := &tokenClaims{}

	t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return signingKey(cfg), nil
	})
	if err != nil {
		return err
	}
	if !t.Valid {
		return fmt.Errorf("invalid token")
	}

	return nil
}

func getUserIDFromToken(cfg config.IntegratedApp, token string) (string, error) {
	claims := &tokenClaims{}

	t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return signingKey(cfg), nil
	})
	if err != nil {
		return "", err
	}
	if !t.Valid {
		return "", fmt.Errorf("invalid token")
	}

	return claims.User, nil
}
