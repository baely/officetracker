package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/baely/officetracker/pkg/model"
	"github.com/baely/officetracker/testutil/mocks"
)

func TestGetSecret(t *testing.T) {
	mockDB := mocks.NewMockDB()
	service := New(mockDB, nil)

	// First call should generate a new secret
	req := model.GetSecretRequest{
		Meta: model.GetSecretRequestMeta{
			UserID: 1,
		},
	}

	resp, err := service.GetSecret(req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Secret)
	assert.Greater(t, len(resp.Secret), 10, "Secret should be reasonably long")

	// Verify secret was saved to DB
	assert.Len(t, mockDB.SaveSecretCalls, 1)
	assert.Equal(t, 1, mockDB.SaveSecretCalls[0].UserID)
	assert.Equal(t, resp.Secret, mockDB.SaveSecretCalls[0].Secret)
}

func TestGetSecret_ErrorHandling(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnSaveSecret = assert.AnError
	service := New(mockDB, nil)

	req := model.GetSecretRequest{
		Meta: model.GetSecretRequestMeta{
			UserID: 1,
		},
	}

	_, err := service.GetSecret(req)
	assert.Error(t, err)
}
