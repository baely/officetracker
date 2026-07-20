package server

import (
	"strings"
	"testing"

	"github.com/baely/officetracker/internal/embed"
)

// The settings template renders the export section in both modes.
func TestSettingsPageExportSection(t *testing.T) {
	for _, standalone := range []bool{false, true} {
		var sb strings.Builder
		page := settingsPage{basePage: basePage{IsLoggedIn: true, IsStandalone: standalone}}
		if err := embed.Settings.Execute(&sb, page); err != nil {
			t.Fatalf("execute settings template (standalone=%v): %v", standalone, err)
		}
		if !strings.Contains(sb.String(), `href="/api/v1/export/officetracker-data.zip"`) {
			t.Errorf("missing export download link (standalone=%v)", standalone)
		}
	}
}
