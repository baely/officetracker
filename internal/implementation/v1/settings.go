package v1

import "github.com/baely/officetracker/pkg/model"

func (i *Service) GetSettings(req model.GetSettingsRequest) (model.GetSettingsResponse, error) {
	accounts, err := i.db.GetUserGithubAccounts(req.Meta.UserID)
	if err != nil {
		return model.GetSettingsResponse{}, err
	}
	return model.GetSettingsResponse{
		GithubAccounts: accounts,
	}, nil
}
