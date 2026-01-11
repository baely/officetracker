package v1

import (
	"fmt"
	"strings"
	"time"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/pkg/model"
)

func (i *Service) PostSecret(req model.PostSecretRequest) (model.PostSecretResponse, error) {
	name := strings.TrimSpace(req.Data.Name)
	if name == "" {
		return model.PostSecretResponse{}, fmt.Errorf("token name cannot be empty")
	}

	secret := auth.GenerateSecret()
	err := i.db.SaveSecret(req.Meta.UserID, secret, name)
	if err != nil {
		return model.PostSecretResponse{}, err
	}

	return model.PostSecretResponse{
		Secret: secret,
	}, nil
}

func (i *Service) ListTokens(req model.ListTokensRequest) (model.ListTokensResponse, error) {
	tokens, err := i.db.ListActiveTokens(req.Meta.UserID)
	if err != nil {
		return model.ListTokensResponse{}, err
	}

	var tokenInfos []model.TokenInfo
	for _, token := range tokens {
		tokenInfos = append(tokenInfos, model.TokenInfo{
			TokenID:   token.TokenID,
			Name:      token.Name,
			CreatedAt: token.CreatedAt.Format(time.RFC3339),
		})
	}

	return model.ListTokensResponse{
		Tokens: tokenInfos,
	}, nil
}

func (i *Service) RevokeToken(req model.RevokeTokenRequest) (model.RevokeTokenResponse, error) {
	err := i.db.RevokeToken(req.Meta.UserID, req.Meta.TokenID)
	if err != nil {
		return model.RevokeTokenResponse{Success: false}, err
	}

	return model.RevokeTokenResponse{Success: true}, nil
}
