package auth

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/util"
)

type Auth struct {
	baseUri string

	auth0Cfg *config.Auth0
	ghCfg    *config.Github

	db       database.Databaser
	redis    *database.Redis
	provider *oidc.Provider
}

func NewAuth(cfg config.AppConfigurer, db database.Databaser, redis *database.Redis) (*Auth, error) {
	appCfg, ok := cfg.(config.IntegratedApp)
	if !ok {
		return nil, nil
	}

	var provider *oidc.Provider
	var err error

	// Skip Auth0 initialization for dummy/test credentials
	if appCfg.Auth0.ClientID != "" && appCfg.Auth0.ClientID != "auth0_client_id" && appCfg.Auth0.ClientID != "dummy_client_id" {
		provider, err = oidc.NewProvider(
			context.Background(),
			"https://"+appCfg.Auth0.Domain+"/",
		)
		if err != nil {
			return nil, err
		}
	}

	return &Auth{
		baseUri:  util.BaseUri(appCfg),
		auth0Cfg: &appCfg.Auth0,
		ghCfg:    &appCfg.Github,

		db:       db,
		redis:    redis,
		provider: provider,
	}, nil
}
