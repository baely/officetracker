package util

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/baely/officetracker/internal/config"
)

func TestQualifiedDomain(t *testing.T) {
	tests := []struct {
		name   string
		cfg    config.Domain
		want   string
	}{
		{
			name: "domain without subdomain",
			cfg: config.Domain{
				Domain:    "example.com",
				Subdomain: "",
			},
			want: "example.com",
		},
		{
			name: "domain with subdomain",
			cfg: config.Domain{
				Domain:    "example.com",
				Subdomain: "app",
			},
			want: "app.example.com",
		},
		{
			name: "domain with nested subdomain",
			cfg: config.Domain{
				Domain:    "example.com",
				Subdomain: "dev.app",
			},
			want: "dev.app.example.com",
		},
		{
			name: "localhost without subdomain",
			cfg: config.Domain{
				Domain:    "localhost",
				Subdomain: "",
			},
			want: "localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := QualifiedDomain(tt.cfg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBaseUri(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.IntegratedApp
		want string
	}{
		{
			name: "production domain with https",
			cfg: config.IntegratedApp{
				App: config.App{
					Port: "8080",
				},
				Domain: config.Domain{
					Protocol:  "https",
					Domain:    "example.com",
					Subdomain: "",
					BasePath:  "",
				},
			},
			want: "https://example.com",
		},
		{
			name: "production domain with subdomain",
			cfg: config.IntegratedApp{
				App: config.App{
					Port: "8080",
				},
				Domain: config.Domain{
					Protocol:  "https",
					Domain:    "example.com",
					Subdomain: "app",
					BasePath:  "",
				},
			},
			want: "https://app.example.com",
		},
		{
			name: "production domain with base path",
			cfg: config.IntegratedApp{
				App: config.App{
					Port: "8080",
				},
				Domain: config.Domain{
					Protocol:  "https",
					Domain:    "example.com",
					Subdomain: "",
					BasePath:  "/api",
				},
			},
			want: "https://example.com/api",
		},
		{
			name: "localhost with port",
			cfg: config.IntegratedApp{
				App: config.App{
					Port: "3000",
				},
				Domain: config.Domain{
					Protocol:  "http",
					Domain:    "localhost",
					Subdomain: "",
					BasePath:  "",
				},
			},
			want: "http://localhost:3000",
		},
		{
			name: "localhost with port and base path",
			cfg: config.IntegratedApp{
				App: config.App{
					Port: "3000",
				},
				Domain: config.Domain{
					Protocol:  "http",
					Domain:    "localhost",
					Subdomain: "",
					BasePath:  "/v1",
				},
			},
			want: "http://localhost:3000/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BaseUri(tt.cfg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBasePath(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Domain
		want string
	}{
		{
			name: "empty base path",
			cfg: config.Domain{
				BasePath: "",
			},
			want: "",
		},
		{
			name: "root path",
			cfg: config.Domain{
				BasePath: "/",
			},
			want: "/",
		},
		{
			name: "api base path",
			cfg: config.Domain{
				BasePath: "/api",
			},
			want: "/api",
		},
		{
			name: "nested base path",
			cfg: config.Domain{
				BasePath: "/api/v1",
			},
			want: "/api/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BasePath(tt.cfg)
			assert.Equal(t, tt.want, got)
		})
	}
}
