package auth_test

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/baely/officetracker/internal/auth"
	"github.com/baely/officetracker/internal/config"
)

var defaultDomainCfg = config.Domain{
	Protocol:  "https",
	Subdomain: "www",
	Domain:    "example.com",
	BasePath:  "/",
}

func TestSSOUri(t *testing.T) {
	t.Run("demo", func(t *testing.T) {
		cfg := config.IntegratedApp{
			App: config.App{
				Demo: true,
			},
		}

		got := auth.SSOUri(cfg)

		require.Equal(t, "/auth/demo", got)
	})

	t.Run("oauth", func(t *testing.T) {
		cfg := config.IntegratedApp{
			App: config.App{
				Port: "80",
				Demo: false,
			},
			Domain: defaultDomainCfg,
			Github: config.Github{
				ClientID: "client_id",
				Secret:   "secret",
			},
		}

		got := auth.SSOUri(cfg)

		require.Equal(t, "https://github.com/login/oauth/authorize?client_id=client_id&redirect_uri=https%3A%2F%2Fwww.example.com%2Fauth%2Fcallback%2Fgithub&response_type=code&scope=read%3Auser&state=state", got)
	})
}

func TestClearCookie(t *testing.T) {
	t.Run("clear_cookie", func(t *testing.T) {
		cfg := config.IntegratedApp{
			App: config.App{
				Port: "80",
				Demo: false,
			},
			Domain: defaultDomainCfg,
		}

		rec := httptest.NewRecorder()
		auth.ClearCookie(cfg, rec)

		cookie := rec.Result().Cookies()[0]

		require.Equal(t, "user", cookie.Name)
		require.Equal(t, "", cookie.Value)
		require.Equal(t, "/", cookie.Path)
		require.Equal(t, "www.example.com", cookie.Domain)
		require.Equal(t, false, cookie.Secure)
		require.Equal(t, true, cookie.HttpOnly)
		require.Equal(t, "1970-01-01 00:00:00 +0000 UTC", cookie.Expires.String())
	})
}
