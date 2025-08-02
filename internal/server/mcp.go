package server

import (
	"github.com/go-chi/chi/v5"

	"github.com/baely/officetracker/internal/auth"
	v1 "github.com/baely/officetracker/internal/implementation/v1"
)

func mcpRouter(service *v1.Service) func(chi.Router) {
	middlewares := chi.Middlewares{AllowedAuthMethods(auth.MethodSSO, auth.MethodSecret, auth.MethodExcluded)}
	return func(r chi.Router) {
		r.With(middlewares...).Handle("/", service.McpHandler())
	}
}
