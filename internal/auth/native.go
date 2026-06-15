package auth

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
)

type nativeExchangeRequest struct {
	IDToken string `json:"id_token"`
}

type nativeExchangeResponse struct {
	Token string `json:"token"`
}

// HandleNativeExchange swaps a native-app Auth0 ID token for a long-lived API
// secret (MethodSecret), used by mobile clients in place of the web cookie flow.
func (a *Auth) HandleNativeExchange(cfg config.IntegratedApp, db database.Databaser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if a.nativeClientID == "" {
			slog.Error("native auth requested but AUTH0_NATIVE_CLIENT_ID is not configured")
			http.Error(w, "native auth not configured", http.StatusNotImplemented)
			return
		}

		var req nativeExchangeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IDToken == "" {
			http.Error(w, "missing id_token", http.StatusBadRequest)
			return
		}

		verifier := a.provider.Verifier(&oidc.Config{ClientID: a.nativeClientID})
		idToken, err := verifier.Verify(ctx, req.IDToken)
		if err != nil {
			slog.Warn(fmt.Sprintf("native id token verification failed: %v", err))
			http.Error(w, "invalid id_token", http.StatusUnauthorized)
			return
		}

		var profile Profile
		if err := idToken.Claims(&profile); err != nil || profile.Sub == "" {
			slog.Error(fmt.Sprintf("failed to parse native id token claims: %v", err))
			http.Error(w, "invalid id_token", http.StatusUnauthorized)
			return
		}

		userID, err := subjectToUserID(db, profile)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to get/create user: %v", err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if err := a.updateLoginForUser(userID, profile); err != nil {
			slog.Warn(fmt.Sprintf("failed to update native profile: %v", err))
		}

		suspended, err := db.IsUserSuspended(userID)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to check suspension: %v", err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if suspended {
			http.Error(w, "account suspended", http.StatusForbidden)
			return
		}

		secret := GenerateSecret()
		if err := db.SaveSecret(userID, secret, "Office Tracker mobile app"); err != nil {
			slog.Error(fmt.Sprintf("failed to save native secret: %v", err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		slog.Info("native auth exchange succeeded", "userID", userID)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(nativeExchangeResponse{Token: secret}); err != nil {
			slog.Error(fmt.Sprintf("failed to write native exchange response: %v", err))
		}
	}
}
