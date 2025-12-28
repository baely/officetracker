package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/baely/officetracker/pkg/model"
	"github.com/baely/officetracker/testutil/mocks"
)

func TestHealthCheck(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	req := model.HealthCheckRequest{}
	resp, err := service.Healthcheck(req)

	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
}

func TestValidateAuth(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	req := model.ValidateAuthRequest{
		Meta: model.ValidateAuthRequestMeta{
			UserID: 1,
		},
	}

	resp, err := service.ValidateAuth(req)

	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
}
