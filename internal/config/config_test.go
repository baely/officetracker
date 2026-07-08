package config

import (
	"os"
	"testing"
)

// LoadIntegratedApp reads configuration purely from environment variables via
// envconfig, joining nested struct prefixes with underscores. Verify the full
// set of nested prefixes maps as expected.
func TestLoadIntegratedApp(t *testing.T) {
	t.Setenv("APP_ENV", "cloud")
	t.Setenv("APP_PORT", "8080")
	t.Setenv("DOMAIN_PROTOCOL", "https")
	t.Setenv("DOMAIN_SUBDOMAIN", "app")
	t.Setenv("DOMAIN_DOMAIN", "officetracker.com.au")
	t.Setenv("DOMAIN_BASE_PATH", "/x")
	t.Setenv("POSTGRES_HOST", "db.internal")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "ot")
	t.Setenv("POSTGRES_PASSWORD", "secret")
	t.Setenv("POSTGRES_DBNAME", "officetracker")
	t.Setenv("REDIS_HOST", "redis.internal")
	t.Setenv("REDIS_DB", "3")
	t.Setenv("AUTH0_DOMAIN", "example.auth0.com")
	t.Setenv("AUTH0_CLIENT_ID", "cid")
	t.Setenv("AUTH0_CLIENT_SECRET", "csecret")
	t.Setenv("AUTH0_NATIVE_CLIENT_ID", "native-cid")
	t.Setenv("SIGNING_KEY", "super-secret-key")

	cfg, err := LoadIntegratedApp()
	if err != nil {
		t.Fatalf("LoadIntegratedApp: %v", err)
	}

	checks := []struct {
		name      string
		got, want any
	}{
		{"App.Env", cfg.App.Env, "cloud"},
		{"App.Port", cfg.App.Port, "8080"},
		{"Domain.Protocol", cfg.Domain.Protocol, "https"},
		{"Domain.Subdomain", cfg.Domain.Subdomain, "app"},
		{"Domain.Domain", cfg.Domain.Domain, "officetracker.com.au"},
		{"Domain.BasePath", cfg.Domain.BasePath, "/x"},
		{"Postgres.Host", cfg.Postgres.Host, "db.internal"},
		{"Postgres.DBName", cfg.Postgres.DBName, "officetracker"},
		{"Redis.Host", cfg.Redis.Host, "redis.internal"},
		{"Redis.DB", cfg.Redis.DB, 3},
		{"Auth0.Domain", cfg.Auth0.Domain, "example.auth0.com"},
		{"Auth0.NativeClientID", cfg.Auth0.NativeClientID, "native-cid"},
		{"SigningKey", cfg.SigningKey, "super-secret-key"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}

	// GetApp satisfies the AppConfigurer interface.
	var configurer AppConfigurer = cfg
	if configurer.GetApp().Env != "cloud" {
		t.Errorf("GetApp().Env = %q, want cloud", configurer.GetApp().Env)
	}
}

// A clean environment loads successfully with zero values — envconfig has no
// required fields here. Numeric fields must be genuinely unset (not empty
// string), otherwise envconfig fails parsing "" as an int.
func TestLoadIntegratedAppEmpty(t *testing.T) {
	for _, k := range []string{
		"APP_ENV", "APP_PORT", "DOMAIN_PROTOCOL", "DOMAIN_SUBDOMAIN", "DOMAIN_DOMAIN",
		"DOMAIN_BASE_PATH", "POSTGRES_HOST", "AUTH0_DOMAIN", "SIGNING_KEY", "REDIS_DB",
	} {
		orig, had := os.LookupEnv(k)
		os.Unsetenv(k)
		if had {
			t.Cleanup(func() { os.Setenv(k, orig) })
		}
	}
	cfg, err := LoadIntegratedApp()
	if err != nil {
		t.Fatalf("LoadIntegratedApp with clean env: %v", err)
	}
	if cfg.App.Env != "" || cfg.SigningKey != "" || cfg.Redis.DB != 0 {
		t.Errorf("expected zero-value config, got %+v", cfg)
	}
}

func TestStandaloneGetApp(t *testing.T) {
	cfg := StandaloneApp{App: App{Env: "standalone", Port: "9000"}}
	if cfg.GetApp().Port != "9000" {
		t.Errorf("StandaloneApp.GetApp().Port = %q, want 9000", cfg.GetApp().Port)
	}
	var _ AppConfigurer = cfg
}
