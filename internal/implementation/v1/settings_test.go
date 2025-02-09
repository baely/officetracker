package v1_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	v1 "github.com/baely/officetracker/internal/implementation/v1"
	"github.com/baely/officetracker/internal/database/mocks"
	"github.com/baely/officetracker/pkg/model"
)

func TestService_GetSettings(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name           string
		userID         int
		mockSetup      func(db *mocks.MockDatabaser)
		expectedResult model.GetSettingsResponse
		expectedError  error
	}{
		{
			name:   "success with github accounts",
			userID: 123,
			mockSetup: func(db *mocks.MockDatabaser) {
				db.EXPECT().
					GetUserGithubAccounts(123).
					Return([]string{"user1", "user2"}, nil)
			},
			expectedResult: model.GetSettingsResponse{
				GithubAccounts: []string{"user1", "user2"},
			},
			expectedError: nil,
		},
		{
			name:   "success with no github accounts",
			userID: 456,
			mockSetup: func(db *mocks.MockDatabaser) {
				db.EXPECT().
					GetUserGithubAccounts(456).
					Return([]string{}, nil)
			},
			expectedResult: model.GetSettingsResponse{
				GithubAccounts: []string{},
			},
			expectedError: nil,
		},
		{
			name:   "database error",
			userID: 789,
			mockSetup: func(db *mocks.MockDatabaser) {
				db.EXPECT().
					GetUserGithubAccounts(789).
					Return(nil, errors.New("database error"))
			},
			expectedResult: model.GetSettingsResponse{},
			expectedError: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := mocks.NewMockDatabaser(ctrl)
			tt.mockSetup(mockDB)

			service := v1.New(mockDB, nil)
			result, err := service.GetSettings(model.GetSettingsRequest{
				Meta: model.GetSettingsRequestMeta{
					UserID: tt.userID,
				},
			})

			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, result)
			}
		})
	}
}