package auth

import (
	"context"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/util"
	"github.com/coreos/go-oidc/v3/oidc"
)

type Auth struct {
	baseUri string

	auth0Cfg *config.Auth0
	ghCfg    *config.Github

	redis    *database.Redis
	provider *oidc.Provider
}

func NewAuth(cfg config.AppConfigurer, redis *database.Redis) (*Auth, error) {
	appCfg, ok := cfg.(config.IntegratedApp)
	if !ok {
		return nil, nil
	}

	provider, err := oidc.NewProvider(
		context.Background(),
		"https://"+appCfg.Auth0.Domain+"/",
	)
	if err != nil {
		return nil, err
	}

	return &Auth{
		baseUri:  util.BaseUri(appCfg),
		auth0Cfg: &appCfg.Auth0,
		ghCfg:    &appCfg.Github,

		redis:    redis,
		provider: provider,
	}, nil
}
