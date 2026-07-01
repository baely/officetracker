package stats

import (
	"context"
	"fmt"

	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/pkg/model"
)

// TrackedDaysCollector reports the all-time total number of tracked days
// (entries with a non-Untracked state). This is the app's headline "value"
// metric and is fully aggregate/non-identifiable.
type TrackedDaysCollector struct {
	DB database.Databaser
}

func (c TrackedDaysCollector) Name() string { return "tracked_days" }

func (c TrackedDaysCollector) Collect(_ context.Context) ([]model.StatWidget, error) {
	count, err := c.DB.CountTrackedDays()
	if err != nil {
		return nil, fmt.Errorf("count tracked days: %w", err)
	}
	return []model.StatWidget{{
		Key:   "tracked_days_total",
		Title: "Total Days Tracked",
		Value: formatInt(count),
		Unit:  "days",
		Group: "Usage",
		Order: 10,
	}}, nil
}

// AttendanceSplitCollector reports the percentage breakdown of tracked days
// across the work states (WFH / Office / Other). Aggregate only.
type AttendanceSplitCollector struct {
	DB database.Databaser
}

func (c AttendanceSplitCollector) Name() string { return "attendance_split" }

func (c AttendanceSplitCollector) Collect(_ context.Context) ([]model.StatWidget, error) {
	counts, err := c.DB.CountEntriesByState()
	if err != nil {
		return nil, fmt.Errorf("count entries by state: %w", err)
	}

	total := 0
	for _, n := range counts {
		total += n
	}
	if total == 0 {
		return nil, nil
	}

	// Fixed presentation order and labels for the meaningful work states.
	type stateInfo struct {
		state model.State
		key   string
		title string
		order int
	}
	states := []stateInfo{
		{model.StateWorkFromOffice, "attendance_office_pct", "Office %", 20},
		{model.StateWorkFromHome, "attendance_home_pct", "Work From Home %", 21},
		{model.StateOther, "attendance_other_pct", "Other %", 22},
	}

	var widgets []model.StatWidget
	for _, s := range states {
		pct := float64(counts[s.state]) / float64(total) * 100
		widgets = append(widgets, model.StatWidget{
			Key:   s.key,
			Title: s.title,
			Value: fmt.Sprintf("%.0f", pct),
			Unit:  "%",
			Group: "Attendance",
			Order: s.order,
		})
	}
	return widgets, nil
}
