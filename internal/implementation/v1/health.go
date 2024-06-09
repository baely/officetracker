package v1

import "github.com/baely/officetracker/pkg/model"

func Healthcheck(req model.HealthCheckRequest) (model.HealthCheckResponse, error) {
	return model.HealthCheckResponse{}, nil
}

func ValidateAuth(req model.ValidateAuthRequest) (model.ValidateAuthResponse, error) {
	return model.ValidateAuthResponse{}, nil
}
