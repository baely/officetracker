package server

import (
	"strings"
	"testing"

	"github.com/baely/officetracker/internal/embed"
)

// The form and settings pages are now static: they carry no per-request data and
// fetch everything client-side. These tests ensure the templates still compose
// (base + page) and render with a nil data context, which is exactly how they
// are rendered at server startup.

// TestFormTemplateRendersStatic ensures the form template composes and renders
// its default (client-overridable) tracking start month.
func TestFormTemplateRendersStatic(t *testing.T) {
	var buf strings.Builder
	if err := embed.Form.Execute(&buf, nil); err != nil {
		t.Fatalf("failed to execute form template: %v", err)
	}
	// html/template's JS escaper pads injected values with spaces; collapse
	// whitespace before asserting.
	normalised := strings.Join(strings.Fields(buf.String()), " ")
	if !strings.Contains(normalised, "trackingStartMonth = 10") {
		t.Fatalf("expected rendered form to default trackingStartMonth to 10")
	}
	if !strings.Contains(buf.String(), `id="calendar"`) {
		t.Fatalf("expected rendered form to contain the calendar element")
	}
}

// TestSettingsTemplateRendersStatic ensures the settings template composes and
// renders with a nil data context.
func TestSettingsTemplateRendersStatic(t *testing.T) {
	var buf strings.Builder
	if err := embed.Settings.Execute(&buf, nil); err != nil {
		t.Fatalf("failed to execute settings template: %v", err)
	}
	if !strings.Contains(buf.String(), `id="year-start-month"`) {
		t.Fatalf("expected rendered settings to contain the tracking-year selector")
	}
}
