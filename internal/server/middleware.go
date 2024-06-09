package server

import (
	"fmt"
	"net/http"
	"slices"
)

func AllowedAuthMethods(authMethods ...AuthMethod) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authMethod, err := getAuthMethod(r)
			if err != nil {
				err = fmt.Errorf("failed to get auth method: %w", err)
				http.Error(w, internalErrorMsg, http.StatusInternalServerError)
				return
			}

			if !slices.Contains(authMethods, authMethod) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
