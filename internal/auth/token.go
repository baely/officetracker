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
	User   int      `json:"user"`
	Scopes []string `json:"scopes"`
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
		ClearCookie(cfg, w)
		return 0
	}

	return userID
}

func GetScopes(r *http.Request) ([]string, error) {
	cookie, err := r.Cookie(userCookie)
	if err != nil {
		return nil, err
	}

	scopes, err := getScopesFromToken(config.IntegratedApp{}, cookie.Value)
	if err != nil {
		err = fmt.Errorf("failed to get scopes: %w", err)
		return nil, err
	}

	return scopes, nil
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

func generateToken(cfg config.IntegratedApp, userID int, scopes []string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user":   userID,
		"scopes": scopes,
	})

	tokenString, err := token.SignedString(signingKey(cfg))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func IssueToken(cfg config.IntegratedApp, w http.ResponseWriter, userID int, scopes []string) error {
	token, err := generateToken(cfg, userID, scopes)
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
	slog.Info(fmt.Sprintf("Issuing cookie for user %d", userID))
	http.SetCookie(w, &cookie)

	return nil
}

func validUser(db database.Databaser, cfg config.IntegratedApp, token string) (int, error) {
	userID, err := getUserIDFromToken(cfg, token)
	if err != nil {
		err = fmt.Errorf("failed to get user id from token: %w", err)
		return 0, err
	}

	// If user ID is low, assume it is "new" user ID
	if userID < 100 {
		return userID, nil
	}

	_, _, err = db.GetUser(userID)
	if err != nil {
		err = fmt.Errorf("failed to get user: %w", err)
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

func getScopesFromToken(cfg config.IntegratedApp, token string) ([]string, error) {
	claims := &tokenClaims{}

	t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return signingKey(cfg), nil
	})
	if err != nil {
		return nil, err
	}
	if !t.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims.Scopes, nil
}
