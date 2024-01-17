package auth

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/baely/officetracker/internal/util"
)

const (
	userCookie = "user"
)

var (
	loginExpiration = time.Hour * 24 * 30
)

type tokenClaims struct {
	jwt.RegisteredClaims
	User string `json:"user"`
}

func signingKey() []byte {
	return []byte(os.Getenv("SIGNING_KEY"))
}

func GetUserID(r *http.Request) string {
	cookie, err := r.Cookie(userCookie)
	if err != nil {
		return ""
	}

	userID, err := getUserIDFromToken(cookie.Value)
	if err != nil {
		return ""
	}

	return userID
}

func generateToken(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": userID,
	})

	tokenString, err := token.SignedString(signingKey())
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func issueToken(w http.ResponseWriter, userID string) error {
	token, err := generateToken(userID)
	if err != nil {
		return err
	}

	domain := util.QualifiedDomain()
	if domain == "localhost" {
		domain = ""
	}

	cookie := http.Cookie{
		Name:     userCookie,
		Value:    token,
		Path:     util.BasePath(),
		Expires:  time.Now().Add(loginExpiration),
		Domain:   util.QualifiedDomain(),
		HttpOnly: true,
		Secure:   false,
	}
	http.SetCookie(w, &cookie)

	slog.Info(fmt.Sprintf("issued token: %+v", cookie))

	return nil
}

func validateToken(token string) error {
	claims := &tokenClaims{}

	t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return signingKey(), nil
	})
	if err != nil {
		return err
	}
	if !t.Valid {
		return fmt.Errorf("invalid token")
	}

	return nil
}

func getUserIDFromToken(token string) (string, error) {
	claims := &tokenClaims{}

	t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return signingKey(), nil
	})
	if err != nil {
		return "", err
	}
	if !t.Valid {
		return "", fmt.Errorf("invalid token")
	}

	return claims.User, nil
}
