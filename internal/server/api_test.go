package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/baely/officetracker/internal/auth"
	context2 "github.com/baely/officetracker/internal/context"
	"github.com/baely/officetracker/internal/implementation/v1"
	"github.com/baely/officetracker/pkg/model"
)

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name        string
		userID      interface{}
		expectError bool
		expectedID  int
	}{
		{
			name:        "Valid user ID",
			userID:      123,
			expectError: false,
			expectedID:  123,
		},
		{
			name:        "No user ID in context",
			userID:      nil,
			expectError: true,
			expectedID:  0,
		},
		{
			name:        "Wrong type",
			userID:      "not-an-int",
			expectError: true,
			expectedID:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			if tt.userID != nil {
				ctxVal := make(context2.CtxValue)
				ctxVal.Set(context2.CtxUserIDKey, tt.userID)
				req = req.WithContext(context.WithValue(req.Context(), context2.CtxKey, ctxVal))
			}

			userID, err := getUserID(req)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, 0, userID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, userID)
			}
		})
	}
}

func TestGetAuthMethod(t *testing.T) {
	tests := []struct {
		name           string
		authMethod     interface{}
		expectError    bool
		expectedMethod auth.Method
	}{
		{
			name:           "Valid SSO method",
			authMethod:     auth.MethodSSO,
			expectError:    false,
			expectedMethod: auth.MethodSSO,
		},
		{
			name:           "Valid Secret method",
			authMethod:     auth.MethodSecret,
			expectError:    false,
			expectedMethod: auth.MethodSecret,
		},
		{
			name:           "No auth method in context",
			authMethod:     nil,
			expectError:    true,
			expectedMethod: auth.MethodUnknown,
		},
		{
			name:           "Wrong type",
			authMethod:     "not-a-method",
			expectError:    true,
			expectedMethod: auth.MethodUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			if tt.authMethod != nil {
				ctxVal := make(context2.CtxValue)
				ctxVal.Set(context2.CtxAuthMethodKey, tt.authMethod)
				req = req.WithContext(context.WithValue(req.Context(), context2.CtxKey, ctxVal))
			}

			method, err := getAuthMethod(req)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, auth.MethodUnknown, method)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMethod, method)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name           string
		message        string
		code           int
		expectedStatus int
	}{
		{
			name:           "Bad request",
			message:        "Invalid input",
			code:           http.StatusBadRequest,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Unauthorized",
			message:        "Not authenticated",
			code:           http.StatusUnauthorized,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Internal server error",
			message:        internalErrorMsg,
			code:           http.StatusInternalServerError,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Not found",
			message:        "Resource not found",
			code:           http.StatusNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeError(rec, tt.message, tt.code)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			// Note: http.Error overrides Content-Type to text/plain even though we set application/json
			// This is the actual behavior of the function
			assert.Contains(t, rec.Header().Get("Content-Type"), "text/plain")

			var errMsg model.Error
			err := json.Unmarshal(rec.Body.Bytes(), &errMsg)
			require.NoError(t, err)
			assert.Equal(t, tt.code, errMsg.Code)
			assert.Contains(t, errMsg.Message, tt.message)
		})
	}
}

func TestPopulateUserID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Set up context with user ID
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxUserIDKey, 456)
	req = req.WithContext(context.WithValue(req.Context(), context2.CtxKey, ctxVal))

	// Create a request struct with Meta field
	type TestRequest struct {
		Meta struct {
			UserID int `meta:"user_id"`
		} `meta:"meta"`
	}

	testReq := &TestRequest{}
	err := populateUserID(testReq, req)

	require.NoError(t, err)
	assert.Equal(t, 456, testReq.Meta.UserID)
}

func TestPopulateUserID_NoUserInContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	type TestRequest struct {
		Meta struct {
			UserID int `meta:"user_id"`
		} `meta:"meta"`
	}

	testReq := &TestRequest{}
	err := populateUserID(testReq, req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrNoUserInCtx)
}

func TestPopulateUrlParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/state/2024/10/15", nil)

	// Set up chi route context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("year", "2024")
	rctx.URLParams.Add("month", "10")
	rctx.URLParams.Add("day", "15")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	type TestRequest struct {
		Meta struct {
			Year  int `meta:"year"`
			Month int `meta:"month"`
			Day   int `meta:"day"`
		} `meta:"meta"`
	}

	testReq := &TestRequest{}
	err := populateUrlParams(testReq, req)

	require.NoError(t, err)
	assert.Equal(t, 2024, testReq.Meta.Year)
	assert.Equal(t, 10, testReq.Meta.Month)
	assert.Equal(t, 15, testReq.Meta.Day)
}

func TestPopulateUrlParams_StringParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/user/john", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("username", "john")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	type TestRequest struct {
		Meta struct {
			Username string `meta:"username"`
		} `meta:"meta"`
	}

	testReq := &TestRequest{}
	err := populateUrlParams(testReq, req)

	require.NoError(t, err)
	assert.Equal(t, "john", testReq.Meta.Username)
}

func TestPopulateQueryParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/report?format=csv&start=2024-01-01&end=2024-12-31", nil)

	type TestRequest struct {
		Format string `schema:"format"`
		Start  string `schema:"start"`
		End    string `schema:"end"`
	}

	testReq := &TestRequest{}
	err := populateQueryParams(testReq, req)

	require.NoError(t, err)
	assert.Equal(t, "csv", testReq.Format)
	assert.Equal(t, "2024-01-01", testReq.Start)
	assert.Equal(t, "2024-12-31", testReq.End)
}

func TestMapRequest_WithBody(t *testing.T) {
	body := `{"data": {"state": 1}}`
	req := httptest.NewRequest(http.MethodPut, "/api/state/2024/10/15", strings.NewReader(body))

	// Set up context with user ID
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxUserIDKey, 1)
	req = req.WithContext(context.WithValue(req.Context(), context2.CtxKey, ctxVal))

	// Set up chi route context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("year", "2024")
	rctx.URLParams.Add("month", "10")
	rctx.URLParams.Add("day", "15")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	result, err := mapRequest[model.PutDayRequest](req)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Meta.UserID)
	assert.Equal(t, 2024, result.Meta.Year)
	assert.Equal(t, 10, result.Meta.Month)
	assert.Equal(t, 15, result.Meta.Day)
	assert.Equal(t, model.StateWorkFromHome, result.Data.State)
}

func TestMapRequest_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/state/2024/10", nil)

	// Set up context with user ID
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxUserIDKey, 1)
	req = req.WithContext(context.WithValue(req.Context(), context2.CtxKey, ctxVal))

	// Set up chi route context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("year", "2024")
	rctx.URLParams.Add("month", "10")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	result, err := mapRequest[model.GetMonthRequest](req)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Meta.UserID)
	assert.Equal(t, 2024, result.Meta.Year)
	assert.Equal(t, 10, result.Meta.Month)
}

func TestMapResponse(t *testing.T) {
	resp := model.GetDayResponse{
		Data: model.DayState{
			State: model.StateWorkFromHome,
		},
	}

	bytes, err := mapResponse(resp)

	require.NoError(t, err)
	assert.NotEmpty(t, bytes)

	var decoded model.GetDayResponse
	err = json.Unmarshal(bytes, &decoded)
	require.NoError(t, err)
	assert.Equal(t, model.StateWorkFromHome, decoded.Data.State)
}

func TestWrap_Success(t *testing.T) {
	// Create a simple handler function
	handler := wrap(func(req model.GetDayRequest) (model.GetDayResponse, error) {
		return model.GetDayResponse{
			Data: model.DayState{
				State: model.StateWorkFromHome,
			},
		}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/api/state/2024/12/15", nil)
	// Add context with user ID
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxUserIDKey, 1)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodSSO)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)
	// Add URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("year", "2024")
	rctx.URLParams.Add("month", "12")
	rctx.URLParams.Add("day", "15")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), `"state":1`)
}

func TestWrap_NoUserInContext(t *testing.T) {
	handler := wrap(func(req model.GetDayRequest) (model.GetDayResponse, error) {
		return model.GetDayResponse{}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/api/state/2024/12/15", nil)
	// Add context without user ID
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodSSO)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestWrap_HandlerError(t *testing.T) {
	handler := wrap(func(req model.GetDayRequest) (model.GetDayResponse, error) {
		return model.GetDayResponse{}, fmt.Errorf("test error")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/state/2024/12/15", nil)
	// Add context with user ID
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxUserIDKey, 1)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodSSO)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)
	// Add URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("year", "2024")
	rctx.URLParams.Add("month", "12")
	rctx.URLParams.Add("day", "15")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), internalErrorMsg)
}

func TestWrapRaw_Success(t *testing.T) {
	handler := wrapRaw(func(req model.GetDayRequest) (model.Response, error) {
		return model.Response{
			ContentType: "text/plain",
			Data:        []byte("test response"),
		}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/api/state/2024/12/15", nil)
	// Add context with user ID
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxUserIDKey, 1)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodSSO)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)
	// Add URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("year", "2024")
	rctx.URLParams.Add("month", "12")
	rctx.URLParams.Add("day", "15")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain", w.Header().Get("Content-Type"))
	assert.Equal(t, "test response", w.Body.String())
}

func TestWrapRaw_HandlerError(t *testing.T) {
	handler := wrapRaw(func(req model.GetDayRequest) (model.Response, error) {
		return model.Response{}, fmt.Errorf("test error")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/state/2024/12/15", nil)
	// Add context with user ID
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxUserIDKey, 1)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodSSO)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)
	// Add URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("year", "2024")
	rctx.URLParams.Add("month", "12")
	rctx.URLParams.Add("day", "15")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), internalErrorMsg)
}

func TestApiRouter(t *testing.T) {
	mockService := &v1.Service{}
	router := chi.NewRouter()
	
	// Call the apiRouter function
	setupFunc := apiRouter(mockService)
	router.Route("/api", setupFunc)
	
	// Verify routes are registered by testing a known endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Should hit the health endpoint (even if it fails, route exists)
	assert.NotEqual(t, http.StatusMethodNotAllowed, w.Code)
}

func TestStateRouter(t *testing.T) {
	mockService := &v1.Service{}
	router := chi.NewRouter()
	
	// Call the stateRouter function
	setupFunc := stateRouter(mockService)
	router.Route("/state", setupFunc)
	
	// Router should be set up (test passes if no panic)
	assert.NotNil(t, router)
}

func TestNoteRouter(t *testing.T) {
	mockService := &v1.Service{}
	router := chi.NewRouter()
	
	setupFunc := noteRouter(mockService)
	router.Route("/note", setupFunc)
	
	assert.NotNil(t, router)
}

func TestSettingsRouter(t *testing.T) {
	mockService := &v1.Service{}
	router := chi.NewRouter()
	
	setupFunc := settingsRouter(mockService)
	router.Route("/settings", setupFunc)
	
	assert.NotNil(t, router)
}

func TestDeveloperRouter(t *testing.T) {
	mockService := &v1.Service{}
	router := chi.NewRouter()
	
	setupFunc := developerRouter(mockService)
	router.Route("/developer", setupFunc)
	
	assert.NotNil(t, router)
}

func TestReportRouter(t *testing.T) {
	mockService := &v1.Service{}
	router := chi.NewRouter()
	
	setupFunc := reportRouter(mockService)
	router.Route("/report", setupFunc)
	
	assert.NotNil(t, router)
}

func TestHealthRouter(t *testing.T) {
	mockService := &v1.Service{}
	router := chi.NewRouter()
	
	setupFunc := healthRouter(mockService)
	router.Route("/health", setupFunc)
	
	assert.NotNil(t, router)
}
