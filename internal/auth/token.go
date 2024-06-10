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

func GetUserID(db database.Databaser, cfg config.IntegratedApp, w http.ResponseWriter, r *http.Request) int {
	cookie, err := r.Cookie(userCookie)
	if err != nil {
		return 0
	}

	userID, err := validUser(db, cfg, cookie.Value)
	if err != nil {
		err = fmt.Errorf("invalid user: %w", err)
		slog.Error(err.Error())
		ClearCookie(w)
		return 0
	}

	return userID
}

func GetUserFromSecret(db database.Databaser, r *http.Request) int {
	secret := r.Header.Get("Authorization")
	if secret == "" {
		return 0
	}
	if !strings.HasPrefix(secret, "Bearer ") {
		slog.Error("invalid secret format")
		return 0
	}
	secret = strings.TrimPrefix(secret, "Bearer ")
	userID, err := db.GetUserBySecret(secret)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to get user id from secret: %v", err))
		return 0
	}
	return userID
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
	http.SetCookie(w, &cookie)

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

func validUser(db database.Databaser, cfg config.IntegratedApp, token string) (int, error) {
	userID, err := getUserIDFromToken(cfg, token)
	if err != nil {
		return 0, err
	}

	_, err = db.GetUser(userID)
	if err != nil {
		return 0, err
	}

	return userID, nil
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
