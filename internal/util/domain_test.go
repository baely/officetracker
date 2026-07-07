package util

import (
	"testing"

	"github.com/baely/officetracker/internal/config"
)

func TestQualifiedDomain(t *testing.T) {
	cases := []struct {
		name string
		cfg  config.Domain
		want string
	}{
		{"no subdomain", config.Domain{Domain: "officetracker.com.au"}, "officetracker.com.au"},
		{"with subdomain", config.Domain{Subdomain: "app", Domain: "officetracker.com.au"}, "app.officetracker.com.au"},
		{"localhost", config.Domain{Domain: "localhost"}, "localhost"},
		{"empty subdomain not prefixed", config.Domain{Subdomain: "", Domain: "example.com"}, "example.com"},
	}
	for _, c := range cases {
		if got := QualifiedDomain(c.cfg); got != c.want {
			t.Errorf("%s: QualifiedDomain = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestBaseUri(t *testing.T) {
	cases := []struct {
		name string
		cfg  config.IntegratedApp
		want string
	}{
		{
			name: "production https with subdomain",
			cfg: config.IntegratedApp{
				Domain: config.Domain{Protocol: "https", Subdomain: "app", Domain: "officetracker.com.au"},
			},
			want: "https://app.officetracker.com.au",
		},
		{
			name: "localhost injects port",
			cfg: config.IntegratedApp{
				App:    config.App{Port: "8080"},
				Domain: config.Domain{Protocol: "http", Domain: "localhost"},
			},
			want: "http://localhost:8080",
		},
		{
			name: "base path appended",
			cfg: config.IntegratedApp{
				Domain: config.Domain{Protocol: "https", Domain: "example.com", BasePath: "/tracker"},
			},
			want: "https://example.com/tracker",
		},
		{
			name: "non-localhost does not get port",
			cfg: config.IntegratedApp{
				App:    config.App{Port: "8080"},
				Domain: config.Domain{Protocol: "https", Domain: "example.com"},
			},
			want: "https://example.com",
		},
	}
	for _, c := range cases {
		if got := BaseUri(c.cfg); got != c.want {
			t.Errorf("%s: BaseUri = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestBasePath(t *testing.T) {
	if got := BasePath(config.Domain{BasePath: "/x"}); got != "/x" {
		t.Errorf("BasePath = %q, want /x", got)
	}
	if got := BasePath(config.Domain{}); got != "" {
		t.Errorf("BasePath empty = %q, want empty", got)
	}
}

// NormaliseStartMonth clamps out-of-range values to the October default and
// leaves valid months untouched. This underpins every tracking-year query.
func TestNormaliseStartMonth(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{0, 10},  // below range -> default
		{-5, 10}, // negative -> default
		{13, 10}, // above range -> default
		{1, 1},   // January boundary
		{10, 10}, // October
		{12, 12}, // December boundary
		{7, 7},   // arbitrary valid
	}
	for _, c := range cases {
		if got := NormaliseStartMonth(c.in); got != c.want {
			t.Errorf("NormaliseStartMonth(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}
