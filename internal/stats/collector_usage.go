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
// UsageStats bundles the usage metrics derived from the request log so they can
// be fetched together in a single query.
type UsageStats struct {
	MAU      int
	AvgDAU   float64
	Requests int
}

type UsageQuerier interface {
	// Usage returns MAU, average DAU and request count over the trailing 30
	// days. Implementations compute all three in one pass.
	Usage(ctx context.Context) (UsageStats, error)
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

	u, err := c.Querier.Usage(ctx)
	if err != nil {
		return nil, fmt.Errorf("usage: %w", err)
	}
	c.lastMAU = u.MAU

	return []model.StatWidget{
		{
			Key:   "mau_30d",
			Title: "Monthly Active Users",
			Value: formatInt(u.MAU),
			Unit:  "users",
			Group: "Usage (30d)",
			Order: 1,
		},
		{
			Key:   "avg_dau_30d",
			Title: "Avg Daily Active Users",
			Value: fmt.Sprintf("%.1f", u.AvgDAU),
			Unit:  "users",
			Group: "Usage (30d)",
			Order: 2,
		},
		{
			Key:   "requests_30d",
			Title: "Authenticated Requests",
			Value: formatInt(u.Requests),
			Unit:  "req",
			Group: "Usage (30d)",
			Order: 3,
		},
	}, nil
}

// MAU returns the most recently collected MAU value, or 0 if unknown.
func (c *UsageCollector) MAU() int { return c.lastMAU }
