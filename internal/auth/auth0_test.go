package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/baely/officetracker/testutil/mocks"
)

func TestParseAuth0Subject(t *testing.T) {
	tests := []struct {
		name         string
		sub          string
		wantProvider string
		wantID       string
		wantErr      bool
	}{
		{
			name:         "github provider",
			sub:          "github|12345",
			wantProvider: "github",
			wantID:       "12345",
			wantErr:      false,
		},
		{
			name:         "google provider",
			sub:          "google-oauth2|67890",
			wantProvider: "google-oauth2",
			wantID:       "67890",
			wantErr:      false,
		},
		{
			name:         "auth0 native",
			sub:          "auth0|abc123",
			wantProvider: "auth0",
			wantID:       "abc123",
			wantErr:      false,
		},
		{
			name:    "invalid format - no pipe",
			sub:     "invalid",
			wantErr: true,
		},
		{
			name:    "invalid format - too many parts",
			sub:     "github|12345|extra",
			wantErr: true,
		},
		{
			name:    "empty string",
			sub:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, id, err := parseAuth0Subject(tt.sub)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid subject format")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantProvider, provider)
				assert.Equal(t, tt.wantID, id)
			}
		})
	}
}

func TestAddLoginToUser(t *testing.T) {
	mockDB := mocks.NewMockDB()
	auth := &Auth{
		db: mockDB,
	}

	profile := Profile{
		Sub:      "github|12345",
		Nickname: "testuser",
		Picture:  "https://example.com/avatar.jpg",
	}

	err := auth.addLoginToUser(1, profile)
	require.NoError(t, err)

	// Verify LinkAuth0Account was called
	assert.Len(t, mockDB.LinkAuth0AccountCalls, 1)
	call := mockDB.LinkAuth0AccountCalls[0]
	assert.Equal(t, 1, call.UserID)
	assert.Equal(t, "github|12345", call.Sub)
	assert.Contains(t, call.Profile, "github|12345")
	assert.Contains(t, call.Profile, "testuser")
}

func TestAddLoginToUser_Error(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnLinkAuth0Account = assert.AnError
	auth := &Auth{
		db: mockDB,
	}

	profile := Profile{
		Sub: "github|12345",
	}

	err := auth.addLoginToUser(1, profile)
	assert.Error(t, err)
}

func TestUpdateLoginForUser(t *testing.T) {
	mockDB := mocks.NewMockDB()
	auth := &Auth{
		db: mockDB,
	}

	// Set up existing auth0 user
	mockDB.AddAuth0User("github|12345", 1)

	profile := Profile{
		Sub:      "github|12345",
		Nickname: "updateduser",
		Picture:  "https://example.com/new-avatar.jpg",
	}

	err := auth.updateLoginForUser(1, profile)
	require.NoError(t, err)

	// Verify UpdateAuth0Profile was called
	assert.Len(t, mockDB.UpdateAuth0ProfileCalls, 1)
	call := mockDB.UpdateAuth0ProfileCalls[0]
	assert.Equal(t, "github|12345", call.Sub)
	assert.Contains(t, call.Profile, "github|12345")
	assert.Contains(t, call.Profile, "updateduser")
}

func TestUpdateLoginForUser_Error(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnUpdateAuth0Profile = assert.AnError
	auth := &Auth{
		db: mockDB,
	}

	profile := Profile{
		Sub: "github|12345",
	}

	err := auth.updateLoginForUser(1, profile)
	assert.Error(t, err)
}

func TestSubjectToUserID_Tier1_ExistingAuth0User(t *testing.T) {
	mockDB := mocks.NewMockDB()

	// Set up existing auth0 user
	mockDB.AddAuth0User("github|12345", 42)

	profile := Profile{
		Sub:      "github|12345",
		Nickname: "testuser",
	}

	userID, err := subjectToUserID(mockDB, profile)
	require.NoError(t, err)
	assert.Equal(t, 42, userID)

	// Verify profile was updated
	assert.Len(t, mockDB.UpdateAuth0ProfileCalls, 1)
}

func TestSubjectToUserID_Tier2_GithubMigration(t *testing.T) {
	mockDB := mocks.NewMockDB()

	// Set up existing GitHub user
	mockDB.AddGitHubUser("12345", 99)

	profile := Profile{
		Sub:      "github|12345",
		Nickname: "testuser",
	}

	userID, err := subjectToUserID(mockDB, profile)
	require.NoError(t, err)
	assert.Equal(t, 99, userID)

	// Verify GitHub user was migrated to Auth0
	assert.Len(t, mockDB.LinkAuth0AccountCalls, 1)
	call := mockDB.LinkAuth0AccountCalls[0]
	assert.Equal(t, 99, call.UserID)
	assert.Equal(t, "github|12345", call.Sub)
}

func TestSubjectToUserID_Tier3_NewUser(t *testing.T) {
	mockDB := mocks.NewMockDB()

	profile := Profile{
		Sub:      "google-oauth2|67890",
		Nickname: "newuser",
	}

	userID, err := subjectToUserID(mockDB, profile)
	require.NoError(t, err)
	assert.NotZero(t, userID)

	// Verify new user was created
	assert.Len(t, mockDB.SaveUserByAuth0SubCalls, 1)
	call := mockDB.SaveUserByAuth0SubCalls[0]
	assert.Equal(t, "google-oauth2|67890", call.Sub)
}

func TestSubjectToUserID_NonGithubProvider(t *testing.T) {
	mockDB := mocks.NewMockDB()

	profile := Profile{
		Sub:      "auth0|abc123",
		Nickname: "auth0user",
	}

	userID, err := subjectToUserID(mockDB, profile)
	require.NoError(t, err)
	assert.NotZero(t, userID)

	// Should skip tier 2 (GitHub migration) and go straight to tier 3
	assert.Len(t, mockDB.SaveUserByAuth0SubCalls, 1)
	assert.Len(t, mockDB.GetUserByGHIDCalls, 0)
}

func TestSubjectToUserID_InvalidSubject(t *testing.T) {
	mockDB := mocks.NewMockDB()

	profile := Profile{
		Sub:      "invalid",
		Nickname: "testuser",
	}

	_, err := subjectToUserID(mockDB, profile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid subject format")
}

func TestSubjectToUserID_MigrationError(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnLinkAuth0Account = assert.AnError

	// Set up existing GitHub user
	mockDB.AddGitHubUser("12345", 99)

	profile := Profile{
		Sub:      "github|12345",
		Nickname: "testuser",
	}

	_, err := subjectToUserID(mockDB, profile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to migrate github user to auth0")
}
