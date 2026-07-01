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
		Title: "Days Tracked (lifetime)",
		Value: formatInt(count),
		Unit:  "days",
		Group: "Usage (30d)",
		Order: 10,
	}}, nil
}

// AverageOfficeAttendanceCollector reports the share of home/office days spent
// in the office: office / (home + office). "Other" days are excluded as they
// don't represent a home-vs-office choice. Aggregate only.
type AverageOfficeAttendanceCollector struct {
	DB database.Databaser
}

func (c AverageOfficeAttendanceCollector) Name() string { return "avg_office_attendance" }

func (c AverageOfficeAttendanceCollector) Collect(_ context.Context) ([]model.StatWidget, error) {
	counts, err := c.DB.CountEntriesByState()
	if err != nil {
		return nil, fmt.Errorf("count entries by state: %w", err)
	}

	home := counts[model.StateWorkFromHome]
	office := counts[model.StateWorkFromOffice]
	if home+office == 0 {
		return nil, nil
	}
	officePct := int(float64(office)/float64(home+office)*100 + 0.5)

	return []model.StatWidget{{
		Key:   "avg_office_attendance",
		Title: "Average Office Attendance (lifetime)",
		Value: fmt.Sprintf("%d", officePct),
		Unit:  "%",
		Group: "Usage (30d)",
		Order: 11,
	}}, nil
}
