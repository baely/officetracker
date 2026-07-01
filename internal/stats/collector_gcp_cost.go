package stats

import (
	"context"
	"fmt"

	"github.com/baely/officetracker/pkg/model"
)

// CostQuerier abstracts the GCP billing data source (BigQuery billing export).
type CostQuerier interface {
	// GCPCost returns total GCP cost over the trailing 30 days.
	GCPCost(ctx context.Context) (float64, error)
}

// GCPCostCollector emits the GCP cost widget. Skipped if no querier configured.
type GCPCostCollector struct {
	Querier CostQuerier

	// lastCost caches the computed GCP cost so the derived cost-per-user
	// collector can reuse it instead of re-running the (billed) BigQuery query.
	// available is set once Collect has populated it. Pointer receivers are
	// required so the cache survives the Collect call.
	lastCost  float64
	available bool
}

func (c *GCPCostCollector) Name() string { return "gcp_cost" }

func (c *GCPCostCollector) Collect(ctx context.Context) ([]model.StatWidget, error) {
	if c.Querier == nil {
		return nil, nil
	}
	cost, err := c.Querier.GCPCost(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcp cost: %w", err)
	}
	c.lastCost = cost
	c.available = true
	return []model.StatWidget{{
		Key:    "cost_gcp_30d",
		Title:  "GCP Cost",
		Value:  fmt.Sprintf("%.2f", cost),
		Prefix: "$",
		Group:  "Cost (30d)",
		Order:  40,
	}}, nil
}

// Cost returns the most recently collected GCP cost and whether it was
// populated (false if BigQuery is unconfigured or the query failed).
func (c *GCPCostCollector) Cost() (float64, bool) { return c.lastCost, c.available }

// CostPerUserProvider supplies the inputs for the derived cost-per-active-user
// metric. It is satisfied by combining the usage and cost queriers.
type CostPerUserProvider struct {
	Cost  *GCPCostCollector
	Usage *UsageCollector
	Fixed FixedCostConfig
}

// CostPerUserCollector emits total 30d cost / MAU. It relies on the usage
// collector having already run (to reuse its MAU), so it must be registered
// after the usage collector.
type CostPerUserCollector struct {
	Provider CostPerUserProvider
}

func (c CostPerUserCollector) Name() string { return "cost_per_user" }

func (c CostPerUserCollector) Collect(ctx context.Context) ([]model.StatWidget, error) {
	if c.Provider.Usage == nil {
		return nil, nil
	}
	mau := c.Provider.Usage.MAU()
	if mau <= 0 {
		return nil, nil
	}

	var gcp float64
	if c.Provider.Cost != nil {
		// Reuse the cost already computed by GCPCostCollector (registered
		// first) rather than re-running the billed billing-export query.
		gcp, _ = c.Provider.Cost.Cost()
	}

	total := gcp + c.Provider.Fixed.Supabase + c.Provider.Fixed.Redis + c.Provider.Fixed.Auth0
	perUser := total / float64(mau)

	return []model.StatWidget{{
		Key:    "cost_per_active_user_30d",
		Title:  "Cost per Active User",
		Value:  fmt.Sprintf("%.3f", perUser),
		Prefix: "$",
		Group:  "Cost (30d)",
		Order:  44,
	}}, nil
}
