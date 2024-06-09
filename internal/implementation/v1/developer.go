package v1

import (
	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/pkg/model"
)

func (i *implementation) GetSecret(req model.GetSecretRequest) (model.GetSecretResponse, error) {
	secret := auth.GenerateSecret()
	err := i.db.SaveSecret(req.Meta.UserID, secret)
	if err != nil {
		return model.GetSecretResponse{}, err
	}

	return model.GetSecretResponse{
		Secret: secret,
	}, nil
}
