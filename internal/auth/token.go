package auth

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	userCookie = "user"
)

var (
	signingKey      = []byte(os.Getenv("SIGNING_KEY"))
	loginExpiration = time.Hour * 24 * 30
)

type tokenClaims struct {
	jwt.RegisteredClaims
	User string `json:"user"`
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

	tokenString, err := token.SignedString(signingKey)
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

	cookie := http.Cookie{
		Name:     userCookie,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(loginExpiration),
		Domain:   "localhost",
		HttpOnly: true,
		Secure:   true,
	}
	http.SetCookie(w, &cookie)

	return nil
}

func validateToken(token string) error {
	claims := &tokenClaims{}

	t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return signingKey, nil
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
		return signingKey, nil
	})
	if err != nil {
		return "", err
	}
	if !t.Valid {
		return "", fmt.Errorf("invalid token")
	}

	return claims.User, nil
}
