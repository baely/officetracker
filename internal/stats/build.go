package stats

import (
	"context"
	"log/slog"

	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/pkg/model"
)

// bqQuerier satisfies both the usage and cost data-source interfaces. Its
// concrete implementation (BigQuery) is selected at build time: the real adapter
// is compiled in with `-tags bigquery` (bigquery.go), otherwise a no-op
// constructor is used (bigquery_stub.go). This keeps the default build free of
// the heavy GCP SDK.
type bqQuerier interface {
	UsageQuerier
	CostQuerier
}

// BuildRegistry assembles the full collector registry from config and the
// database. DB-backed and fixed-cost collectors always run; BigQuery-backed
// collectors run only when configured and the bigquery adapter is compiled in.
func BuildRegistry(ctx context.Context, cfg Config, db database.Databaser) *Registry {
	r := NewRegistry()

	// Usage collector (may be a no-op if BQ unavailable). Declared up front so
	// the derived cost-per-user collector can reference it.
	usage := &UsageCollector{}

	var costQ CostQuerier
	if cfg.BigQueryEnabled() {
		q, err := newBigQueryQuerier(ctx, cfg)
		if err != nil {
			slog.Error("failed to init bigquery querier; skipping usage/gcp-cost widgets", "error", err.Error())
		} else if q == nil {
			slog.Warn("bigquery configured but adapter not compiled in; build the collector with -tags bigquery")
		} else {
			usage.Querier = q
			costQ = q
		}
	}

	// gcpCost is shared: registered as its own widget collector AND referenced
	// by cost-per-user, so the billed billing-export query runs only once.
	gcpCost := &GCPCostCollector{Querier: costQ}

	// Register in display/dependency order. Usage and gcpCost must precede
	// cost-per-user, which reuses their cached values.
	r.Register(usage)
	r.Register(TrackedDaysCollector{DB: db})
	r.Register(AverageOfficeAttendanceCollector{DB: db})
	r.Register(gcpCost)
	r.Register(FixedCostCollector{Config: cfg.FixedCosts()})
	r.Register(CostPerUserCollector{Provider: CostPerUserProvider{
		Cost:  gcpCost,
		Usage: usage,
		Fixed: cfg.FixedCosts(),
	}})

	return r
}

// Run performs a full collection and persists the snapshot.
func Run(ctx context.Context, cfg Config, db database.Databaser) ([]model.StatWidget, error) {
	r := BuildRegistry(ctx, cfg, db)
	widgets := r.Collect(ctx)
	// Don't persist an empty snapshot: if every collector failed (e.g. a
	// transient DB/BigQuery outage), saving would clobber the last good
	// snapshot and flip the public dashboard to its empty state.
	if len(widgets) == 0 {
		slog.Warn("stats collection produced no widgets; skipping snapshot save")
		return widgets, nil
	}
	if err := db.SaveStatsSnapshot(widgets); err != nil {
		return nil, err
	}
	slog.Info("stats snapshot saved", "widgets", len(widgets))
	return widgets, nil
}
