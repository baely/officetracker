package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegratedApp_GetApp(t *testing.T) {
	cfg := IntegratedApp{
		App: App{
			Env:  "production",
			Port: "8080",
			Demo: false,
		},
	}

	got := cfg.GetApp()
	assert.Equal(t, "production", got.Env)
	assert.Equal(t, "8080", got.Port)
	assert.Equal(t, false, got.Demo)
}

func TestStandaloneApp_GetApp(t *testing.T) {
	cfg := StandaloneApp{
		App: App{
			Env:  "development",
			Port: "3000",
			Demo: true,
		},
	}

	got := cfg.GetApp()
	assert.Equal(t, "development", got.Env)
	assert.Equal(t, "3000", got.Port)
	assert.Equal(t, true, got.Demo)
}

func TestLoadIntegratedApp(t *testing.T) {
	// Set environment variables for testing
	envVars := map[string]string{
		"APP_ENV":              "test",
		"APP_PORT":             "9090",
		"APP_DEMO":             "true",
		"DOMAIN_PROTOCOL":      "https",
		"DOMAIN_SUBDOMAIN":     "test",
		"DOMAIN_DOMAIN":        "example.com",
		"DOMAIN_BASE_PATH":     "/api",
		"POSTGRES_HOST":        "localhost",
		"POSTGRES_PORT":        "5432",
		"POSTGRES_USER":        "testuser",
		"POSTGRES_PASSWORD":    "testpass",
		"POSTGRES_DBNAME":      "testdb",
		"REDIS_HOST":           "localhost:6379",
		"REDIS_USERNAME":       "redisuser",
		"REDIS_PASSWORD":       "redispass",
		"REDIS_DB":             "0",
		"GITHUB_CLIENT_ID":     "github123",
		"GITHUB_SECRET":        "githubsecret",
		"AUTH0_DOMAIN":         "test.auth0.com",
		"AUTH0_CLIENT_ID":      "auth0client",
		"AUTH0_CLIENT_SECRET":  "auth0secret",
		"SIGNING_KEY":          "testsigningkey",
	}

	// Set environment variables
	for k, v := range envVars {
		os.Setenv(k, v)
	}

	// Clean up after test
	defer func() {
		for k := range envVars {
			os.Unsetenv(k)
		}
	}()

	cfg, err := LoadIntegratedApp()
	require.NoError(t, err)

	// Verify App config
	assert.Equal(t, "test", cfg.App.Env)
	assert.Equal(t, "9090", cfg.App.Port)
	assert.Equal(t, true, cfg.App.Demo)

	// Verify Domain config
	assert.Equal(t, "https", cfg.Domain.Protocol)
	assert.Equal(t, "test", cfg.Domain.Subdomain)
	assert.Equal(t, "example.com", cfg.Domain.Domain)
	assert.Equal(t, "/api", cfg.Domain.BasePath)

	// Verify Postgres config
	assert.Equal(t, "localhost", cfg.Postgres.Host)
	assert.Equal(t, "5432", cfg.Postgres.Port)
	assert.Equal(t, "testuser", cfg.Postgres.User)
	assert.Equal(t, "testpass", cfg.Postgres.Password)
	assert.Equal(t, "testdb", cfg.Postgres.DBName)

	// Verify Redis config
	assert.Equal(t, "localhost:6379", cfg.Redis.Host)
	assert.Equal(t, "redisuser", cfg.Redis.Username)
	assert.Equal(t, "redispass", cfg.Redis.Password)
	assert.Equal(t, 0, cfg.Redis.DB)

	// Verify Github config
	assert.Equal(t, "github123", cfg.Github.ClientID)
	assert.Equal(t, "githubsecret", cfg.Github.Secret)

	// Verify Auth0 config
	assert.Equal(t, "test.auth0.com", cfg.Auth0.Domain)
	assert.Equal(t, "auth0client", cfg.Auth0.ClientID)
	assert.Equal(t, "auth0secret", cfg.Auth0.ClientSecret)

	// Verify SigningKey
	assert.Equal(t, "testsigningkey", cfg.SigningKey)
}

func TestLoadIntegratedApp_WithDefaults(t *testing.T) {
	// Clear any existing environment variables
	envVarsToClear := []string{
		"APP_ENV", "APP_PORT", "APP_DEMO",
		"DOMAIN_PROTOCOL", "DOMAIN_SUBDOMAIN", "DOMAIN_DOMAIN", "DOMAIN_BASE_PATH",
		"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DBNAME",
		"REDIS_HOST", "REDIS_USERNAME", "REDIS_PASSWORD", "REDIS_DB",
		"GITHUB_CLIENT_ID", "GITHUB_SECRET",
		"AUTH0_DOMAIN", "AUTH0_CLIENT_ID", "AUTH0_CLIENT_SECRET",
		"SIGNING_KEY",
	}

	for _, k := range envVarsToClear {
		os.Unsetenv(k)
	}

	cfg, err := LoadIntegratedApp()
	require.NoError(t, err)

	// When no env vars are set, config should have zero/empty values
	assert.Equal(t, "", cfg.App.Env)
	assert.Equal(t, "", cfg.App.Port)
	assert.Equal(t, false, cfg.App.Demo)
	assert.Equal(t, "", cfg.Domain.Protocol)
	assert.Equal(t, "", cfg.SigningKey)
}

func TestAppConfigurer_Interface(t *testing.T) {
	// Verify both types implement the interface
	var _ AppConfigurer = (*IntegratedApp)(nil)
	var _ AppConfigurer = (*StandaloneApp)(nil)

	// Test that interface method works
	integrated := IntegratedApp{
		App: App{Env: "integrated", Port: "8080"},
	}
	standalone := StandaloneApp{
		App: App{Env: "standalone", Port: "3000"},
	}

	var cfgIntegrated AppConfigurer = integrated
	var cfgStandalone AppConfigurer = standalone

	assert.Equal(t, "integrated", cfgIntegrated.GetApp().Env)
	assert.Equal(t, "standalone", cfgStandalone.GetApp().Env)
}
