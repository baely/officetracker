package server

import (
	"strings"
	"testing"

	"github.com/baely/officetracker/internal/embed"
	"github.com/baely/officetracker/pkg/model"
)

func day(s model.State) model.DayState { return model.DayState{State: s} }

func TestBuildReportSummary(t *testing.T) {
	// Tracking year 2025 with an October start spans Oct 2024 - Sep 2025.
	state := model.YearState{Months: map[int]model.MonthState{
		10: {Days: map[int]model.DayState{
			1: day(model.StateWorkFromOffice),
			2: day(model.StateWorkFromHome),
			3: day(model.StateScheduledWorkFromOffice),
			4: day(model.StateScheduledWorkFromHome),
			5: day(model.StateOther),     // not a work day
			6: day(model.StateUntracked), // not a work day
		}},
		2: {Days: map[int]model.DayState{
			1: day(model.StateWorkFromOffice),
		}},
		3: {Days: map[int]model.DayState{
			1: day(model.StateOther), // month has no work days: omitted
		}},
	}}

	rows, headline := buildReportSummary(state, 2025, 10)

	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2 (months without work days omitted): %+v", len(rows), rows)
	}
	// Months are ordered by position within the tracking year.
	if rows[0].Month != "October 2024" || rows[0].Present != 2 || rows[0].Total != 4 || rows[0].Percent != "50.00%" {
		t.Errorf("October row = %+v", rows[0])
	}
	if rows[1].Month != "February 2025" || rows[1].Present != 1 || rows[1].Total != 1 || rows[1].Percent != "100.00%" {
		t.Errorf("February row = %+v", rows[1])
	}
	if want := "Present in office for 3 out of 5 days. (60.00%)"; headline != want {
		t.Errorf("headline = %q, want %q", headline, want)
	}
}

func TestBuildReportSummaryEmpty(t *testing.T) {
	rows, headline := buildReportSummary(model.YearState{}, 2025, 10)
	if len(rows) != 0 {
		t.Errorf("rows = %+v, want none", rows)
	}
	if want := "Present in office for 0 out of 0 days. (0.00%)"; headline != want {
		t.Errorf("headline = %q, want %q", headline, want)
	}
}

// TestReportTemplateRenders ensures the report template parses and renders the
// summary rows and export controls.
func TestReportTemplateRenders(t *testing.T) {
	var buf strings.Builder
	err := embed.Report.Execute(&buf, reportPage{
		Year:     2026,
		Rows:     []reportRow{{Month: "October 2025", Present: 2, Total: 4, Percent: "50.00%"}},
		Headline: "Present in office for 2 out of 4 days. (50.00%)",
	})
	if err != nil {
		t.Fatalf("failed to execute report template: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"October 2025", "50.00%", "export-csv", "export-pdf", "report-year"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered report missing %q", want)
		}
	}
}
