package stats

import (
	"context"
	"errors"
	"testing"

	"github.com/baely/officetracker/internal/database/dbtest"
	"github.com/baely/officetracker/pkg/model"
)

type fakeCollector struct {
	name    string
	widgets []model.StatWidget
	err     error
}

func (f fakeCollector) Name() string { return f.name }
func (f fakeCollector) Collect(context.Context) ([]model.StatWidget, error) {
	return f.widgets, f.err
}

// Collect merges widgets from all collectors, orders them by Order, and counts
// (but does not abort on) failures so one broken source can't take down the
// dashboard.
func TestRegistryCollect(t *testing.T) {
	r := NewRegistry()
	r.Register(
		fakeCollector{name: "b", widgets: []model.StatWidget{{Key: "b", Order: 3}}},
		fakeCollector{name: "a", widgets: []model.StatWidget{{Key: "a", Order: 1}}},
		fakeCollector{name: "boom", err: errors.New("kaboom")},
		fakeCollector{name: "c", widgets: []model.StatWidget{{Key: "c", Order: 2}}},
	)

	res := r.Collect(context.Background())

	if res.Failures != 1 {
		t.Errorf("Failures = %d, want 1", res.Failures)
	}
	gotOrder := make([]string, len(res.Widgets))
	for i, w := range res.Widgets {
		gotOrder[i] = w.Key
	}
	want := []string{"a", "c", "b"} // sorted by Order 1,2,3
	if len(gotOrder) != 3 || gotOrder[0] != want[0] || gotOrder[1] != want[1] || gotOrder[2] != want[2] {
		t.Errorf("widget order = %v, want %v", gotOrder, want)
	}
}

// A healthy run persists the snapshot and returns the widgets.
func TestRunPersistsSnapshot(t *testing.T) {
	db := dbtest.New()
	widgets, err := Run(context.Background(), Config{}, db)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(widgets) == 0 {
		t.Fatal("expected some widgets from the default collectors")
	}
	if len(db.SavedSnapshots) != 1 {
		t.Fatalf("expected 1 saved snapshot, got %d", len(db.SavedSnapshots))
	}
	if len(db.SavedSnapshots[0]) != len(widgets) {
		t.Errorf("saved snapshot has %d widgets, want %d", len(db.SavedSnapshots[0]), len(widgets))
	}
}

// A collector failure must NOT overwrite the last good snapshot: the run
// returns the partial widgets but skips the save.
func TestRunSkipsSaveOnFailure(t *testing.T) {
	db := dbtest.New()
	db.Errs = map[string]error{"CountTrackedDays": errors.New("db hiccup")}

	_, err := Run(context.Background(), Config{}, db)
	if err != nil {
		t.Fatalf("Run should swallow collector failures, got %v", err)
	}
	if len(db.SavedSnapshots) != 0 {
		t.Errorf("degraded run should not persist a snapshot, saved %d", len(db.SavedSnapshots))
	}
}

// A failure to persist is surfaced to the caller.
func TestRunSaveError(t *testing.T) {
	db := dbtest.New()
	db.Errs = map[string]error{"SaveStatsSnapshot": errors.New("write failed")}

	if _, err := Run(context.Background(), Config{}, db); err == nil {
		t.Fatal("expected Run to return the snapshot-save error")
	}
}

// Config predicates gate the optional BigQuery-backed collectors.
func TestConfigPredicates(t *testing.T) {
	if (Config{}).BigQueryEnabled() {
		t.Error("empty config should not enable BigQuery")
	}
	full := Config{BQProjectID: "p", BQLogsTable: "d.t"}
	if !full.BigQueryEnabled() {
		t.Error("project+logs table should enable BigQuery")
	}
	if full.BillingEnabled() {
		t.Error("billing needs billing table + project id")
	}
	billing := Config{BQBillingTable: "d.b", BQBillingProjectID: "p"}
	if !billing.BillingEnabled() {
		t.Error("billing table + project id should enable billing")
	}

	fc := Config{CostSupabase: 25, CostRedis: 5, CostAuth0: 0}.FixedCosts()
	if fc.Supabase != 25 || fc.Redis != 5 {
		t.Errorf("FixedCosts = %+v", fc)
	}
}
