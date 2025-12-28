package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/testutil/mocks"
)

func TestGenerateToken(t *testing.T) {
	cfg := config.IntegratedApp{
		SigningKey: "test-signing-key-123",
		Domain: config.Domain{
			Domain:    "example.com",
			Subdomain: "",
			Protocol:  "https",
		},
	}

	userID := 123
	token, err := generateToken(cfg, userID)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify token can be parsed
	claims := &tokenClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(cfg.SigningKey), nil
	})
	require.NoError(t, err)
	assert.True(t, parsedToken.Valid)

	// Verify claims
	assert.Equal(t, userID, claims.User)
	assert.Equal(t, fmt.Sprintf("%d", userID), claims.Subject)
	assert.Equal(t, "example.com", claims.Issuer)
	assert.NotNil(t, claims.IssuedAt)
	assert.NotNil(t, claims.ExpiresAt)

	// Verify expiration is 30 days
	expectedExpiration := claims.IssuedAt.Add(30 * 24 * time.Hour)
	assert.Equal(t, expectedExpiration.Unix(), claims.ExpiresAt.Unix())
}

func TestGenerateToken_WithSubdomain(t *testing.T) {
	cfg := config.IntegratedApp{
		SigningKey: "test-key",
		Domain: config.Domain{
			Domain:    "example.com",
			Subdomain: "app",
		},
	}

	token, err := generateToken(cfg, 1)
	require.NoError(t, err)

	claims := &tokenClaims{}
	_, err = jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(cfg.SigningKey), nil
	})
	require.NoError(t, err)

	assert.Equal(t, "app.example.com", claims.Issuer)
}

func TestGetUserIDFromToken_Valid(t *testing.T) {
	cfg := config.IntegratedApp{
		SigningKey: "test-key",
		Domain: config.Domain{
			Domain: "example.com",
		},
	}

	// Generate a valid token
	token, err := generateToken(cfg, 42)
	require.NoError(t, err)

	// Extract user ID
	userID, err := getUserIDFromToken(cfg, token)
	require.NoError(t, err)
	assert.Equal(t, 42, userID)
}

func TestGetUserIDFromToken_Expired(t *testing.T) {
	cfg := config.IntegratedApp{
		SigningKey: "test-key",
		Domain: config.Domain{
			Domain: "example.com",
		},
	}

	// Create an expired token
	pastTime := time.Now().Add(-31 * 24 * time.Hour) // 31 days ago
	claims := tokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "1",
			Issuer:    "example.com",
			IssuedAt:  jwt.NewNumericDate(pastTime),
			ExpiresAt: jwt.NewNumericDate(pastTime.Add(30 * 24 * time.Hour)), // Already expired
		},
		User: 1,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.SigningKey))
	require.NoError(t, err)

	// Try to validate expired token
	userID, err := getUserIDFromToken(cfg, tokenString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token expired")
	assert.Equal(t, 0, userID)
}

func TestGetUserIDFromToken_MissingClaims(t *testing.T) {
	cfg := config.IntegratedApp{
		SigningKey: "test-key",
		Domain: config.Domain{
			Domain: "example.com",
		},
	}

	tests := []struct {
		name   string
		claims tokenClaims
		errMsg string
	}{
		{
			name: "missing issuer",
			claims: tokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "1",
					IssuedAt:  jwt.NewNumericDate(time.Now()),
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
					// Issuer missing
				},
				User: 1,
			},
			errMsg: "missing required iss claim",
		},
		{
			name: "missing subject",
			claims: tokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    "example.com",
					IssuedAt:  jwt.NewNumericDate(time.Now()),
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
					// Subject missing
				},
				User: 1,
			},
			errMsg: "missing required sub claim",
		},
		{
			name: "missing issued at",
			claims: tokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "1",
					Issuer:    "example.com",
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
					// IssuedAt missing
				},
				User: 1,
			},
			errMsg: "missing required iat claim",
		},
		{
			name: "missing expires at",
			claims: tokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:  "1",
					Issuer:   "example.com",
					IssuedAt: jwt.NewNumericDate(time.Now()),
					// ExpiresAt missing
				},
				User: 1,
			},
			errMsg: "missing required exp claim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, tt.claims)
			tokenString, err := token.SignedString([]byte(cfg.SigningKey))
			require.NoError(t, err)

			userID, err := getUserIDFromToken(cfg, tokenString)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
			assert.Equal(t, 0, userID)
		})
	}
}

func TestGetUserIDFromToken_InvalidIssuer(t *testing.T) {
	cfg := config.IntegratedApp{
		SigningKey: "test-key",
		Domain: config.Domain{
			Domain: "example.com",
		},
	}

	// Create token with wrong issuer
	claims := tokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "1",
			Issuer:    "wrong.com", // Wrong issuer
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
		User: 1,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.SigningKey))
	require.NoError(t, err)

	userID, err := getUserIDFromToken(cfg, tokenString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token issuer")
	assert.Equal(t, 0, userID)
}

func TestGetUserIDFromToken_SubjectMismatch(t *testing.T) {
	cfg := config.IntegratedApp{
		SigningKey: "test-key",
		Domain: config.Domain{
			Domain: "example.com",
		},
	}

	// Create token where subject doesn't match user
	claims := tokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "999", // Subject doesn't match User
			Issuer:    "example.com",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
		User: 1,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.SigningKey))
	require.NoError(t, err)

	userID, err := getUserIDFromToken(cfg, tokenString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subject mismatch")
	assert.Equal(t, 0, userID)
}

func TestGetUserIDFromToken_WrongSigningKey(t *testing.T) {
	cfg := config.IntegratedApp{
		SigningKey: "correct-key",
		Domain: config.Domain{
			Domain: "example.com",
		},
	}

	// Create token with different signing key
	wrongCfg := cfg
	wrongCfg.SigningKey = "wrong-key"
	token, err := generateToken(wrongCfg, 1)
	require.NoError(t, err)

	// Try to validate with correct key
	userID, err := getUserIDFromToken(cfg, token)
	assert.Error(t, err)
	assert.Equal(t, 0, userID)
}

func TestGetUserIDFromSecret(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.AddUser(1)
	mockDB.SaveSecret(1, "test-secret-123")

	userID, err := getUserIDFromSecret(mockDB, "test-secret-123")
	require.NoError(t, err)
	assert.Equal(t, 1, userID)
}

func TestGetUserIDFromSecret_NotFound(t *testing.T) {
	mockDB := mocks.NewMockDB()

	userID, err := getUserIDFromSecret(mockDB, "nonexistent-secret")
	assert.Error(t, err)
	assert.Equal(t, 0, userID)
	assert.ErrorIs(t, err, database.ErrNoUser)
}

func TestGetUserID_DemoMode(t *testing.T) {
	cfg := config.IntegratedApp{
		App: config.App{
			Demo: true,
		},
	}

	userID, err := GetUserID(cfg, nil, "", MethodSSO)
	require.NoError(t, err)
	assert.Equal(t, 1, userID) // Demo mode always returns user 1
}

func TestGetUserID_SSO(t *testing.T) {
	cfg := config.IntegratedApp{
		SigningKey: "test-key",
		Domain: config.Domain{
			Domain: "example.com",
		},
	}

	token, err := generateToken(cfg, 42)
	require.NoError(t, err)

	userID, err := GetUserID(cfg, nil, token, MethodSSO)
	require.NoError(t, err)
	assert.Equal(t, 42, userID)
}

func TestGetUserID_Secret(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.SaveSecret(1, "my-secret")

	cfg := config.IntegratedApp{}

	userID, err := GetUserID(cfg, mockDB, "my-secret", MethodSecret)
	require.NoError(t, err)
	assert.Equal(t, 1, userID)
}

func TestGetUserID_None(t *testing.T) {
	cfg := config.IntegratedApp{}

	userID, err := GetUserID(cfg, nil, "", MethodNone)
	require.NoError(t, err)
	assert.Equal(t, 0, userID)
}

func TestCookieName(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.IntegratedApp
		want string
	}{
		{
			name: "empty env defaults to base name",
			cfg: config.IntegratedApp{
				App: config.App{Env: ""},
			},
			want: "user",
		},
		{
			name: "cloud env uses base name",
			cfg: config.IntegratedApp{
				App: config.App{Env: "cloud"},
			},
			want: "user",
		},
		{
			name: "development env",
			cfg: config.IntegratedApp{
				App: config.App{Env: "development"},
			},
			want: "user_development",
		},
		{
			name: "staging env",
			cfg: config.IntegratedApp{
				App: config.App{Env: "staging"},
			},
			want: "user_staging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cookieName(tt.cfg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateDevSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		want   string
	}{
		{
			name:   "valid bearer token",
			secret: "Bearer my-secret-token",
			want:   "my-secret-token",
		},
		{
			name:   "bearer lowercase",
			secret: "bearer my-secret-token",
			want:   "my-secret-token",
		},
		{
			name:   "empty secret",
			secret: "",
			want:   "",
		},
		{
			name:   "no bearer prefix",
			secret: "just-a-token",
			want:   "",
		},
		{
			name:   "bearer only",
			secret: "Bearer ",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateDevSecret(tt.secret)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetAuth_FromCookie(t *testing.T) {
	cfg := config.IntegratedApp{
		App: config.App{Env: ""},
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "user",
		Value: "my-jwt-token",
	})

	token, method := GetAuth(cfg, req)
	assert.Equal(t, "my-jwt-token", token)
	assert.Equal(t, MethodSSO, method)
}

func TestGetAuth_FromHeader(t *testing.T) {
	cfg := config.IntegratedApp{}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer my-secret-token")

	token, method := GetAuth(cfg, req)
	assert.Equal(t, "my-secret-token", token)
	assert.Equal(t, MethodSecret, method)
}

func TestGetAuth_CookieTakesPrecedence(t *testing.T) {
	cfg := config.IntegratedApp{
		App: config.App{Env: ""},
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "user",
		Value: "cookie-token",
	})
	req.Header.Set("Authorization", "Bearer header-token")

	token, method := GetAuth(cfg, req)
	assert.Equal(t, "cookie-token", token)
	assert.Equal(t, MethodSSO, method)
}

func TestGetAuth_NoneFound(t *testing.T) {
	cfg := config.IntegratedApp{}

	req := httptest.NewRequest("GET", "/", nil)

	token, method := GetAuth(cfg, req)
	assert.Equal(t, "", token)
	assert.Equal(t, MethodNone, method)
}

func TestMethodString(t *testing.T) {
	tests := []struct {
		method Method
		want   string
	}{
		{MethodNone, "none"},
		{MethodSSO, "sso"},
		{MethodSecret, "secret"},
		{MethodExcluded, "excluded"},
		{MethodUnknown, "unknown"},
		{Method(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.method.String())
		})
	}
}

func TestIssueToken(t *testing.T) {
	cfg := config.IntegratedApp{
		SigningKey: "test-key",
		Domain: config.Domain{
			Domain:   "localhost",
			BasePath: "/",
		},
		App: config.App{
			Port: "3000",
		},
	}

	w := httptest.NewRecorder()
	err := issueToken(cfg, w, 123)
	require.NoError(t, err)

	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "user", cookies[0].Name)
	assert.NotEmpty(t, cookies[0].Value)
	assert.True(t, cookies[0].HttpOnly)

	// Verify the token is valid
	userID, err := getUserIDFromToken(cfg, cookies[0].Value)
	require.NoError(t, err)
	assert.Equal(t, 123, userID)
}

func TestIssueToken_ProductionDomain(t *testing.T) {
	cfg := config.IntegratedApp{
		SigningKey: "test-key",
		Domain: config.Domain{
			Domain:    "example.com",
			Subdomain: "app",
			BasePath:  "/api",
		},
	}

	w := httptest.NewRecorder()
	err := issueToken(cfg, w, 1)
	require.NoError(t, err)

	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "/api", cookies[0].Path)
	assert.Equal(t, "app.example.com", cookies[0].Domain)
}
