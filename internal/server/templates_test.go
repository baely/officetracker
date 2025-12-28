package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/baely/officetracker/internal/auth"
	context2 "github.com/baely/officetracker/internal/context"
	"context"
)

func TestServeHero(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/hero", nil)
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodExcluded)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)
	
	w := httptest.NewRecorder()

	serveHero(w, req, heroPage{})

	// serveHero redirects to /login
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

func TestServeTos(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/tos", nil)
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodExcluded)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)
	
	w := httptest.NewRecorder()

	serveTos(w, req, tosPage{})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestServePrivacy(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/privacy", nil)
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodExcluded)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)
	
	w := httptest.NewRecorder()

	servePrivacy(w, req, privacyPage{})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestServeSuspended(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/suspended", nil)
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodExcluded)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)
	
	w := httptest.NewRecorder()

	serveSuspended(w, req, suspendedPage{})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestServeDeveloper(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/developer", nil)
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodExcluded)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)
	
	w := httptest.NewRecorder()

	serveDeveloper(w, req, developerPage{})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestGetBasePageData(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodSSO)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)

	baseData := getBasePageData(req)

	assert.True(t, baseData.IsLoggedIn)
	assert.False(t, baseData.IsStandalone)
}

func TestGetBasePageData_Standalone(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctxVal := make(context2.CtxValue)
	ctxVal.Set(context2.CtxAuthMethodKey, auth.MethodExcluded)
	ctx := context.WithValue(req.Context(), context2.CtxKey, ctxVal)
	req = req.WithContext(ctx)

	baseData := getBasePageData(req)

	assert.True(t, baseData.IsLoggedIn)
	assert.True(t, baseData.IsStandalone)
}
