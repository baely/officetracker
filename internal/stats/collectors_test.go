package stats

import (
	"context"
	"testing"

	"github.com/baely/officetracker/internal/database/dbtest"
	"github.com/baely/officetracker/pkg/model"
)

type fakeUsageQuerier struct {
	stats UsageStats
	err   error
}

func (f fakeUsageQuerier) Usage(context.Context) (UsageStats, error) { return f.stats, f.err }

type fakeCostQuerier struct {
	cost float64
	err  error
}

func (f fakeCostQuerier) GCPCost(context.Context) (float64, error) { return f.cost, f.err }

func widgetByKey(widgets []model.StatWidget, key string) (model.StatWidget, bool) {
	for _, w := range widgets {
		if w.Key == key {
			return w, true
		}
	}
	return model.StatWidget{}, false
}

// UsageCollector emits MAU / avg DAU / request-count widgets from the querier,
// and caches MAU for the derived cost-per-user metric.
func TestUsageCollector(t *testing.T) {
	c := &UsageCollector{Querier: fakeUsageQuerier{stats: UsageStats{MAU: 1234, AvgDAU: 42.5, Requests: 98765}}}
	widgets, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if w, _ := widgetByKey(widgets, "mau_30d"); w.Value != "1,234" {
		t.Errorf("mau value = %q, want 1,234", w.Value)
	}
	if w, _ := widgetByKey(widgets, "avg_dau_30d"); w.Value != "42.5" {
		t.Errorf("avg dau value = %q, want 42.5", w.Value)
	}
	if w, _ := widgetByKey(widgets, "requests_30d"); w.Value != "98,765" {
		t.Errorf("requests value = %q, want 98,765", w.Value)
	}
	if c.MAU() != 1234 {
		t.Errorf("cached MAU = %d, want 1234", c.MAU())
	}
}

// A nil querier means BigQuery isn't configured, so the collector is skipped.
func TestUsageCollectorNilQuerier(t *testing.T) {
	c := &UsageCollector{}
	widgets, err := c.Collect(context.Background())
	if err != nil || widgets != nil {
		t.Errorf("nil-querier Collect = (%v, %v), want (nil, nil)", widgets, err)
	}
}

func TestGCPCostCollector(t *testing.T) {
	c := &GCPCostCollector{Querier: fakeCostQuerier{cost: 123.456}}
	widgets, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	w, ok := widgetByKey(widgets, "cost_gcp_30d")
	if !ok || w.Value != "123.46" || w.Prefix != "$" {
		t.Errorf("gcp cost widget = %+v", w)
	}
	if cost, avail := c.Cost(); !avail || cost != 123.456 {
		t.Errorf("cached cost = (%v, %v)", cost, avail)
	}
}

// TrackedDaysCollector reports the lifetime count of non-untracked entries.
func TestTrackedDaysCollector(t *testing.T) {
	db := dbtest.New()
	db.SaveDay(1, 1, 1, 2024, model.DayState{State: model.StateWorkFromOffice})
	db.SaveDay(1, 2, 1, 2024, model.DayState{State: model.StateWorkFromHome})
	db.SaveDay(1, 3, 1, 2024, model.DayState{State: model.StateUntracked})

	widgets, err := TrackedDaysCollector{DB: db}.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if w, _ := widgetByKey(widgets, "tracked_days_total"); w.Value != "2" {
		t.Errorf("tracked days = %q, want 2", w.Value)
	}
}

// AverageOfficeAttendanceCollector reports office / (home + office) as a
// rounded percentage, excluding "other" days, and is skipped when there is no
// home/office data.
func TestAverageOfficeAttendanceCollector(t *testing.T) {
	db := dbtest.New()
	// 3 office, 1 home -> 75%.
	db.SaveDay(1, 1, 1, 2024, model.DayState{State: model.StateWorkFromOffice})
	db.SaveDay(1, 2, 1, 2024, model.DayState{State: model.StateWorkFromOffice})
	db.SaveDay(1, 3, 1, 2024, model.DayState{State: model.StateWorkFromOffice})
	db.SaveDay(1, 4, 1, 2024, model.DayState{State: model.StateWorkFromHome})
	db.SaveDay(1, 5, 1, 2024, model.DayState{State: model.StateOther}) // excluded

	widgets, err := AverageOfficeAttendanceCollector{DB: db}.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if w, _ := widgetByKey(widgets, "avg_office_attendance"); w.Value != "75" {
		t.Errorf("office attendance = %q, want 75", w.Value)
	}

	// No home/office data -> no widget.
	empty := dbtest.New()
	widgets, err = AverageOfficeAttendanceCollector{DB: empty}.Collect(context.Background())
	if err != nil || widgets != nil {
		t.Errorf("empty attendance = (%v, %v), want (nil, nil)", widgets, err)
	}
}

func TestFixedCostCollector(t *testing.T) {
	widgets, err := FixedCostCollector{Config: FixedCostConfig{Supabase: 25, Redis: 5, Auth0: 0}}.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(widgets) != 3 {
		t.Fatalf("got %d widgets, want 3", len(widgets))
	}
	if w, _ := widgetByKey(widgets, "cost_supabase_30d"); w.Value != "25.00" || w.Prefix != "$" {
		t.Errorf("supabase widget = %+v", w)
	}
}

// CostPerUserCollector divides total (GCP + fixed) cost by MAU, and is skipped
// when MAU or cost is unavailable.
func TestCostPerUserCollector(t *testing.T) {
	usage := &UsageCollector{Querier: fakeUsageQuerier{stats: UsageStats{MAU: 100}}}
	usage.Collect(context.Background()) // populate cached MAU

	gcp := &GCPCostCollector{Querier: fakeCostQuerier{cost: 50}}
	gcp.Collect(context.Background()) // populate cached cost

	c := CostPerUserCollector{Provider: CostPerUserProvider{
		Cost:  gcp,
		Usage: usage,
		Fixed: FixedCostConfig{Supabase: 30, Redis: 20, Auth0: 0}, // +50 fixed
	}}
	widgets, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	// total = 50 gcp + 50 fixed = 100; per user = 100 / 100 = 1.000
	if w, _ := widgetByKey(widgets, "cost_per_active_user_30d"); w.Value != "1.000" {
		t.Errorf("cost per user = %q, want 1.000", w.Value)
	}

	// Zero MAU -> skipped.
	zeroUsage := &UsageCollector{Querier: fakeUsageQuerier{stats: UsageStats{MAU: 0}}}
	zeroUsage.Collect(context.Background())
	c2 := CostPerUserCollector{Provider: CostPerUserProvider{Cost: gcp, Usage: zeroUsage}}
	if widgets, _ := c2.Collect(context.Background()); widgets != nil {
		t.Errorf("zero-MAU cost-per-user should be skipped, got %v", widgets)
	}
}
