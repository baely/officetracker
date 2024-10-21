package auth_test

import (
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/config"
	"github.com/baely/officetracker/internal/util/testutils"
)

func TestRouter(t *testing.T) {
	t.Run("router", func(t *testing.T) {
		r := chi.NewRouter()

		cfg := config.IntegratedApp{}

		db := testutils.NewMockDatabase()

		auth.Router(cfg, db)(r)

		var routes []string
		for _, route := range r.Routes() {
			routes = append(routes, route.Pattern)
		}

		require.Contains(t, routes, "/logout")
		require.Contains(t, routes, "/callback/github")
		require.Contains(t, routes, "/demo")

	})
}
