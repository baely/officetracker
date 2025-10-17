package v1

import "github.com/baely/officetracker/pkg/model"

// Healthcheck godoc
//
//	@Summary		Health check
//	@Description	Check if the API is running and healthy
//	@Tags			health
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	model.HealthCheckResponse
//	@Router			/health/check [get]
func (i *Service) Healthcheck(_ model.HealthCheckRequest) (model.HealthCheckResponse, error) {
	return model.HealthCheckResponse{
		Status: "ok",
	}, nil
}

// ValidateAuth godoc
//
//	@Summary		Validate authentication
//	@Description	Validate that the provided authentication credentials are valid
//	@Tags			health
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	model.ValidateAuthResponse
//	@Failure		401	{object}	model.Error
//	@Security		BearerAuth
//	@Router			/health/auth [get]
func (i *Service) ValidateAuth(_ model.ValidateAuthRequest) (model.ValidateAuthResponse, error) {
	return model.ValidateAuthResponse{
		Status: "ok",
	}, nil
}
