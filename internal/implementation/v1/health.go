package v1

import "github.com/baely/officetracker/pkg/model"

func (i *Service) Healthcheck(_ model.HealthCheckRequest) (model.HealthCheckResponse, error) {
	return model.HealthCheckResponse{
		Status: "ok",
	}, nil
}

func (i *Service) ValidateAuth(_ model.ValidateAuthRequest) (model.ValidateAuthResponse, error) {
	return model.ValidateAuthResponse{
		Status: "ok",
	}, nil
}
