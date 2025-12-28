package server

import (
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/baely/officetracker/internal/implementation/v1"
)

func TestMcpRouter(t *testing.T) {
	mockService := &v1.Service{}
	router := chi.NewRouter()
	
	setupFunc := mcpRouter(mockService)
	router.Route("/mcp", setupFunc)
	
	assert.NotNil(t, router)
}
