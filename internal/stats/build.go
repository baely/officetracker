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

	// Register in display/dependency order. Usage must precede cost-per-user.
	r.Register(usage)
	r.Register(TrackedDaysCollector{DB: db})
	r.Register(AttendanceSplitCollector{DB: db})
	r.Register(GCPCostCollector{Querier: costQ})
	r.Register(FixedCostCollector{Config: cfg.FixedCosts()})
	r.Register(CostPerUserCollector{Provider: CostPerUserProvider{
		Cost:  costQ,
		Usage: usage,
		Fixed: cfg.FixedCosts(),
	}})

	return r
}

// Run performs a full collection and persists the snapshot.
func Run(ctx context.Context, cfg Config, db database.Databaser) ([]model.StatWidget, error) {
	r := BuildRegistry(ctx, cfg, db)
	widgets := r.Collect(ctx)
	if err := db.SaveStatsSnapshot(widgets); err != nil {
		return nil, err
	}
	slog.Info("stats snapshot saved", "widgets", len(widgets))
	return widgets, nil
}
