package embed

import (
	"html/template"
	"io"
	"maps"
	"testing"
)

func pageData(extra map[string]any) map[string]any {
	m := map[string]any{
		"IsLoggedIn":   true,
		"IsStandalone": false,
	}
	maps.Copy(m, extra)
	return m
}

// TestTemplatesExecute renders every page template with representative data,
// catching template parse/exec regressions without booting the server.
func TestTemplatesExecute(t *testing.T) {
	cases := []struct {
		name string
		tmpl *template.Template
		data map[string]any
	}{
		{"form", Form, pageData(map[string]any{
			"YearlyState": template.JS(`{"data":{"months":{"6":{"days":{"1":{"state":2}}}}}}`),
			"YearlyNotes": template.JS(`{"data":{"6":{"note":"hi"}}}`),
		})},
		{"hero", Hero, pageData(nil)},
		{"login", Login, pageData(map[string]any{"SSOLink": "https://example.com/sso"})},
		{"settings", Settings, pageData(map[string]any{
			"Auth0AuthURL": "https://example.com/link",
			"LinkedAccounts": []map[string]any{
				{"Provider": "github", "ProviderDisplay": "GitHub", "Nickname": "tester"},
			},
			"ThemePreferences": map[string]any{
				"Theme": "default", "WeatherEnabled": false, "TimeBasedEnabled": false,
			},
			"SchedulePreferences": map[string]any{
				"Monday": 0, "Tuesday": 1, "Wednesday": 2, "Thursday": 3,
				"Friday": 0, "Saturday": 0, "Sunday": 0,
			},
		})},
		{"developer", Developer, pageData(nil)},
		{"tos", Tos, pageData(nil)},
		{"privacy", Privacy, pageData(nil)},
		{"suspended", Suspended, pageData(nil)},
		{"error", Error, pageData(map[string]any{"ErrorMessage": "test message", "StatusCode": 404})},
	}

	for _, c := range cases {
		if err := c.tmpl.Execute(io.Discard, c.data); err != nil {
			t.Errorf("template %s failed to execute: %v", c.name, err)
		}
	}
}
