package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/config"
	context2 "github.com/baely/officetracker/internal/context"
	"github.com/baely/officetracker/testutil/mocks"
)

func TestAllowedAuthMethods_SSOOnly(t *testing.T) {
	middleware := AllowedAuthMethods(auth.MethodSSO)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name           string
		authMethod     auth.Method
		expectedStatus int
	}{
		{
			name:           "SSO allowed",
			authMethod:     auth.MethodSSO,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Secret not allowed",
			authMethod:     auth.MethodSecret,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Unknown not allowed",
			authMethod:     auth.MethodUnknown,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			// Set up context with auth method
			ctxVal := make(context2.CtxValue)
			ctxVal.Set(context2.CtxAuthMethodKey, tt.authMethod)
			req = req.WithContext(context.WithValue(req.Context(), context2.CtxKey, ctxVal))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestAllowedAuthMethods_SecretOnly(t *testing.T) {
	middleware := AllowedAuthMethods(auth.MethodSecret)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Set up context with Secret auth method
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodSecret)
	req = req.WithContext(context.WithValue(req.Context(), context2.CtxKey, ctxVal))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAllowedAuthMethods_ExcludedBypass(t *testing.T) {
	middleware := AllowedAuthMethods(auth.MethodExcluded)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Set Excluded method - should pass
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodExcluded)
	req = req.WithContext(context.WithValue(req.Context(), context2.CtxKey, ctxVal))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAllowedAuthMethods_MultipleAllowed(t *testing.T) {
	middleware := AllowedAuthMethods(auth.MethodSSO, auth.MethodSecret)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name           string
		authMethod     auth.Method
		expectedStatus int
	}{
		{
			name:           "SSO allowed",
			authMethod:     auth.MethodSSO,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Secret allowed",
			authMethod:     auth.MethodSecret,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Excluded not allowed",
			authMethod:     auth.MethodExcluded,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			ctxVal := make(context2.CtxValue)
			ctxVal.Set(context2.CtxAuthMethodKey, tt.authMethod)
			req = req.WithContext(context.WithValue(req.Context(), context2.CtxKey, ctxVal))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestAllowedAuthMethods_NoAuthMethod(t *testing.T) {
	middleware := AllowedAuthMethods(auth.MethodSSO)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// No auth method in context - should return internal server error
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestInjectAuth_Standalone(t *testing.T) {
	mockDB := mocks.NewMockDB()
	cfg := config.StandaloneApp{}

	middleware := injectAuth(mockDB, cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that context has auth method and user ID
		ctxVal := context2.GetCtxValue(r)
		authMethod := ctxVal.Get(context2.CtxAuthMethodKey)
		userID := ctxVal.Get(context2.CtxUserIDKey)

		assert.Equal(t, auth.MethodExcluded, authMethod)
		assert.Equal(t, 1, userID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestInjectAuth_Integrated_NoAuth(t *testing.T) {
	mockDB := mocks.NewMockDB()
	cfg := config.IntegratedApp{
		Domain: config.Domain{Domain: "localhost"},
	}

	middleware := injectAuth(mockDB, cfg)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that context has auth method
		ctxVal := context2.GetCtxValue(r)
		authMethod := ctxVal.Get(context2.CtxAuthMethodKey)

		// When no auth provided, GetAuth returns MethodSecret with empty token
		assert.NotNil(t, authMethod)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestLogRequest(t *testing.T) {
	mockDB := mocks.NewMockDB()
	cfg := config.StandaloneApp{}

	server := &Server{
		db:  mockDB,
		cfg: cfg,
	}

	middleware := server.logRequest
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// Add context with auth method and user ID
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodExcluded)
	ctxVal.Set(context2.CtxUserIDKey, 1)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Verify the request was processed
	assert.Equal(t, http.StatusOK, rec.Code)
}
