package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/config"
	context2 "github.com/baely/officetracker/internal/context"
	"github.com/baely/officetracker/testutil/mocks"
)

func TestHandleHero(t *testing.T) {
	mockDB := mocks.NewMockDB()
	cfg := config.StandaloneApp{}

	server := &Server{
		db:  mockDB,
		cfg: cfg,
	}

	req := httptest.NewRequest(http.MethodGet, "/hero", nil)
	// Add auth method to context
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodExcluded)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	server.handleHero(w, req)

	// Hero redirects to /login with 307
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

func TestHandleTos(t *testing.T) {
	mockDB := mocks.NewMockDB()
	cfg := config.StandaloneApp{}

	server := &Server{
		db:  mockDB,
		cfg: cfg,
	}

	req := httptest.NewRequest(http.MethodGet, "/tos", nil)
	w := httptest.NewRecorder()

	server.handleTos(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
}

func TestHandlePrivacy(t *testing.T) {
	mockDB := mocks.NewMockDB()
	cfg := config.StandaloneApp{}

	server := &Server{
		db:  mockDB,
		cfg: cfg,
	}

	req := httptest.NewRequest(http.MethodGet, "/privacy", nil)
	w := httptest.NewRecorder()

	server.handlePrivacy(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
}

func TestHandleSuspended(t *testing.T) {
	mockDB := mocks.NewMockDB()
	cfg := config.StandaloneApp{}

	server := &Server{
		db:  mockDB,
		cfg: cfg,
	}

	req := httptest.NewRequest(http.MethodGet, "/suspended", nil)
	w := httptest.NewRecorder()

	server.handleSuspended(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
}

func TestHandleNotFound(t *testing.T) {
	mockDB := mocks.NewMockDB()
	cfg := config.StandaloneApp{}

	server := &Server{
		db:  mockDB,
		cfg: cfg,
	}

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	// Add auth method to context
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodExcluded)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	server.handleNotFound(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	// errorPage executes a template which should render HTML
	assert.NotEmpty(t, w.Body.String())
}

func TestHandleLogout_Integrated(t *testing.T) {
	mockDB := mocks.NewMockDB()
	cfg := config.IntegratedApp{
		Domain: config.Domain{
			Domain: "localhost",
		},
	}

	server := &Server{
		db:  mockDB,
		cfg: cfg,
	}

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	w := httptest.NewRecorder()

	server.handleLogout(w, req)

	// Should redirect to home with cleared cookie
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "/", w.Header().Get("Location"))
}

func TestStaticHandler(t *testing.T) {
	r := chi.NewRouter()
	staticHandler(r)

	// Test the registered route
	req := httptest.NewRequest(http.MethodGet, "/github-mark-white.png", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Should serve the image
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
}

func TestHandleDeveloper_SSO(t *testing.T) {
	mockDB := mocks.NewMockDB()
	cfg := config.StandaloneApp{}

	server := &Server{
		db:  mockDB,
		cfg: cfg,
	}

	req := httptest.NewRequest(http.MethodGet, "/developer", nil)
	// Add SSO auth method to context
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodSSO)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	server.handleDeveloper(w, req)

	// Should render developer page
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestHandleDeveloper_NotSSO(t *testing.T) {
	mockDB := mocks.NewMockDB()
	cfg := config.StandaloneApp{}

	server := &Server{
		db:  mockDB,
		cfg: cfg,
	}

	req := httptest.NewRequest(http.MethodGet, "/developer", nil)
	// Add non-SSO auth method to context
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodSecret)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	server.handleDeveloper(w, req)

	// Should redirect to home
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "/", w.Header().Get("Location"))
}

func TestHandleIndex_Standalone(t *testing.T) {
	mockDB := mocks.NewMockDB()
	cfg := config.StandaloneApp{}

	server := &Server{
		db:  mockDB,
		cfg: cfg,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Add context - standalone uses MethodExcluded
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodExcluded)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	server.handleIndex(w, req)

	// Should call handleForm which redirects when no user ID
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "/", w.Header().Get("Location"))
}

func TestHandleIndex_IntegratedLoggedIn(t *testing.T) {
	mockDB := mocks.NewMockDB()
	cfg := config.IntegratedApp{
		Domain: config.Domain{Domain: "localhost"},
	}

	server := &Server{
		db:  mockDB,
		cfg: cfg,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Add context with SSO (logged in method)
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodSSO)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	server.handleIndex(w, req)

	// Should call handleForm which redirects when no user ID
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
}

func TestHandleIndex_IntegratedNotLoggedIn(t *testing.T) {
	mockDB := mocks.NewMockDB()
	cfg := config.IntegratedApp{
		Domain: config.Domain{Domain: "localhost"},
	}

	server := &Server{
		db:  mockDB,
		cfg: cfg,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Add context with Unknown method (not logged in)
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodUnknown)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	server.handleIndex(w, req)

	// Should call handleHero then handleForm (both redirect, handleForm wins)
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "/", w.Header().Get("Location"))
}
