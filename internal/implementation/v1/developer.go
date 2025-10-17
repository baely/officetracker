package v1

import (
	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/pkg/model"
)

// GetSecret godoc
//
//	@Summary		Generate API secret
//	@Description	Generate a new API secret for authentication. Revokes all previously generated secrets.
//	@Tags			developer
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	model.GetSecretResponse
//	@Failure		400	{object}	model.Error
//	@Failure		500	{object}	model.Error
//	@Security		CookieAuth
//	@Router			/developer/secret [get]
func (i *Service) GetSecret(req model.GetSecretRequest) (model.GetSecretResponse, error) {
	secret := auth.GenerateSecret()
	err := i.db.SaveSecret(req.Meta.UserID, secret)
	if err != nil {
		return model.GetSecretResponse{}, err
	}

	return model.GetSecretResponse{
		Secret: secret,
	}, nil
}
