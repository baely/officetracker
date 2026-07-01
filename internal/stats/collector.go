// Package stats implements the modular stats collection pipeline that powers
// the public dashboard. Each statistic ("widget") is produced by a Collector.
// A Registry runs every registered collector, aggregates their widgets, and
// persists a single snapshot via the database. Adding a new widget is as simple
// as writing a new Collector and registering it - no schema or UI changes are
// required.
package stats

import (
	"context"
	"log/slog"
	"sort"

	"github.com/baely/officetracker/pkg/model"
)

// Collector produces zero or more stat widgets. Implementations should be
// self-contained and must not panic; return an error instead. A collector that
// cannot run in the current environment (e.g. missing configuration) should
// return an empty slice and nil error so it is simply skipped.
type Collector interface {
	// Name identifies the collector for logging.
	Name() string
	// Collect gathers the widgets this collector is responsible for.
	Collect(ctx context.Context) ([]model.StatWidget, error)
}

// Registry holds a set of collectors and orchestrates a collection run.
type Registry struct {
	collectors []Collector
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds one or more collectors to the registry.
func (r *Registry) Register(collectors ...Collector) {
	r.collectors = append(r.collectors, collectors...)
}

// CollectResult is the outcome of a collection run: the merged widgets plus the
// number of collectors that failed, so the caller can decide whether the
// snapshot is complete enough to persist.
type CollectResult struct {
	Widgets  []model.StatWidget
	Failures int
}

// Collect runs every registered collector and returns the merged, ordered set
// of widgets along with a count of collectors that errored. A failure in one
// collector is logged and skipped so that a single broken source never takes
// down the whole dashboard, but the count lets the caller avoid persisting a
// degraded snapshot over a good one.
func (r *Registry) Collect(ctx context.Context) CollectResult {
	var widgets []model.StatWidget
	failures := 0
	for _, c := range r.collectors {
		w, err := c.Collect(ctx)
		if err != nil {
			slog.Error("stats collector failed", "collector", c.Name(), "error", err.Error())
			failures++
			continue
		}
		widgets = append(widgets, w...)
	}
	sort.SliceStable(widgets, func(i, j int) bool {
		return widgets[i].Order < widgets[j].Order
	})
	return CollectResult{Widgets: widgets, Failures: failures}
}
