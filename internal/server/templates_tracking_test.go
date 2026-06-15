package server

import (
	"io"
	"strings"
	"testing"

	"github.com/baely/officetracker/internal/embed"
	"github.com/baely/officetracker/pkg/model"
)

// TestFormTemplateRendersTrackingStartMonth ensures the form template parses and
// the injected tracking start month is rendered into the page (form.js consumes it).
func TestFormTemplateRendersTrackingStartMonth(t *testing.T) {
	var buf strings.Builder
	err := embed.Form.Execute(&buf, formPage{
		YearlyState:        "{}",
		YearlyNotes:        "{}",
		TrackingStartMonth: 7,
	})
	if err != nil {
		t.Fatalf("failed to execute form template: %v", err)
	}
	// html/template's JS escaper pads injected values with spaces; collapse
	// whitespace before asserting.
	normalised := strings.Join(strings.Fields(buf.String()), " ")
	if !strings.Contains(normalised, "trackingStartMonth = 7 || 10") {
		t.Fatalf("expected rendered form to inject trackingStartMonth = 7")
	}
}

// TestSettingsTemplateRendersCalendarPreference ensures the settings template
// parses and renders the calendar preference selector value.
func TestSettingsTemplateRendersCalendarPreference(t *testing.T) {
	err := embed.Settings.Execute(io.Discard, settingsPage{
		CalendarPreferences: model.CalendarPreferences{TrackingYearStartMonth: 4},
	})
	if err != nil {
		t.Fatalf("failed to execute settings template: %v", err)
	}
}
