package stats

import (
	"context"
	"fmt"

	"github.com/baely/officetracker/pkg/model"
)

// UsageQuerier abstracts the data source for usage metrics derived from logs.
// The production implementation is backed by BigQuery (see bigquery.go), but the
// interface keeps collectors decoupled and unit-testable, and lets the whole
// usage pipeline be omitted entirely when BigQuery is not configured.
type UsageQuerier interface {
	// MAU returns distinct active users over the trailing 30 days.
	MAU(ctx context.Context) (int, error)
	// AvgDAU returns the mean daily active users over the trailing 30 days.
	AvgDAU(ctx context.Context) (float64, error)
	// RequestCount returns total processed requests over the trailing 30 days.
	RequestCount(ctx context.Context) (int, error)
}

// UsageCollector emits the MAU, average DAU and request-count widgets. It also
// exposes the computed MAU so that a derived collector (cost-per-user) can reuse
// it without re-querying.
type UsageCollector struct {
	Querier UsageQuerier

	// lastMAU caches the most recent MAU value for derived metrics. It is set
	// during Collect. Zero means "unknown".
	lastMAU int
}

func (c *UsageCollector) Name() string { return "usage" }

func (c *UsageCollector) Collect(ctx context.Context) ([]model.StatWidget, error) {
	if c.Querier == nil {
		return nil, nil
	}

	var widgets []model.StatWidget

	mau, err := c.Querier.MAU(ctx)
	if err != nil {
		return nil, fmt.Errorf("mau: %w", err)
	}
	c.lastMAU = mau
	widgets = append(widgets, model.StatWidget{
		Key:   "mau_30d",
		Title: "Monthly Active Users",
		Value: formatInt(mau),
		Unit:  "users",
		Group: "Usage",
		Order: 1,
	})

	avgDAU, err := c.Querier.AvgDAU(ctx)
	if err != nil {
		return nil, fmt.Errorf("avg dau: %w", err)
	}
	widgets = append(widgets, model.StatWidget{
		Key:   "avg_dau_30d",
		Title: "Avg Daily Active Users",
		Value: fmt.Sprintf("%.1f", avgDAU),
		Unit:  "users",
		Group: "Usage",
		Order: 2,
	})

	reqs, err := c.Querier.RequestCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("request count: %w", err)
	}
	widgets = append(widgets, model.StatWidget{
		Key:   "requests_30d",
		Title: "Requests (30d)",
		Value: formatInt(reqs),
		Unit:  "req",
		Group: "Usage",
		Order: 3,
	})

	return widgets, nil
}

// MAU returns the most recently collected MAU value, or 0 if unknown.
func (c *UsageCollector) MAU() int { return c.lastMAU }
