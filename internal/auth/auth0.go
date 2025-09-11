package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

func (a *Auth) Auth0OauthCfg() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     a.auth0Cfg.ClientID,
		ClientSecret: a.auth0Cfg.ClientSecret,
		Endpoint:     a.provider.Endpoint(),
		RedirectURL:  fmt.Sprintf("%sauth/callback/auth0", a.baseUri),
		Scopes:       []string{oidc.ScopeOpenID, "profile"},
	}
}

func (a *Auth) Auth0SSOUri() (string, error) {
	// Generate a secure random state using crypto/rand
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", fmt.Errorf("failed to generate state: %v", err)
	}
	state := base64.URLEncoding.EncodeToString(stateBytes)

	// Store the state in Redis with 0 as userID (new user), expiring in 10 minutes
	key := fmt.Sprintf("auth0:state:%s", state)
	err := a.redis.SetState(context.Background(), key, 0, 10*time.Minute)
	if err != nil {
		return "", fmt.Errorf("failed to store state: %v", err)
	}

	return a.Auth0OauthCfg().AuthCodeURL(state), nil
}

func handleAuth0Callback() http.HandlerFunc {
	return nil
}
