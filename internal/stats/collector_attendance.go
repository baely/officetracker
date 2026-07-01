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
		Title: "Lifetime Days Tracked",
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

	pct := func(s model.State) int {
		return int(float64(counts[s])/float64(total)*100 + 0.5)
	}
	home := pct(model.StateWorkFromHome)
	office := pct(model.StateWorkFromOffice)
	other := pct(model.StateOther)

	// A single combined widget showing the split, e.g. "37% / 58% / 5%" under
	// the label "Home / Office / Other".
	return []model.StatWidget{{
		Key:   "attendance_split",
		Title: "Attendance (Home / Office / Other)",
		Value: fmt.Sprintf("%d%% / %d%% / %d%%", home, office, other),
		Group: "Usage",
		Order: 11,
	}}, nil
}
