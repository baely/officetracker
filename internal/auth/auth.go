package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/util"
)

// SessionStore is the subset of the Redis client the auth package uses to keep
// login state: OAuth state nonces, Auth0-backed sessions and refresh locks.
type SessionStore interface {
	SetState(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	GetState(ctx context.Context, key string) (string, error)
	GetStateInt(ctx context.Context, key string) (int, error)
	SetStateNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	DeleteState(ctx context.Context, key string) error
}

type Auth struct {
	baseUri string

	auth0Cfg *config.Auth0
	ghCfg    *config.Github

	nativeClientID string

	db       database.Databaser
	store    SessionStore
	provider *oidc.Provider
}

func NewAuth(cfg config.AppConfigurer, db database.Databaser, redis *database.Redis) (*Auth, error) {
	appCfg, ok := cfg.(config.IntegratedApp)
	if !ok {
		return nil, nil
	}

	var provider *oidc.Provider
	var err error

	provider, err = oidc.NewProvider(
		context.Background(),
		appCfg.Auth0.Domain,
	)
	if err != nil {
		return nil, err
	}

	return &Auth{
		baseUri:        util.BaseUri(appCfg),
		auth0Cfg:       &appCfg.Auth0,
		ghCfg:          &appCfg.Github,
		nativeClientID: appCfg.Auth0.NativeClientID,

		db:       db,
		store:    redis,
		provider: provider,
	}, nil
}

func ClearCookie(cfg config.IntegratedApp, w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName(cfg),
		Value:    "",
		Path:     util.BasePath(cfg.Domain),
		Expires:  time.Unix(0, 0),
		Domain:   util.QualifiedDomain(cfg.Domain),
		HttpOnly: true,
		Secure:   secureCookies(cfg),
	})
}
