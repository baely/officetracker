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
}

func (c GCPCostCollector) Name() string { return "gcp_cost" }

func (c GCPCostCollector) Collect(ctx context.Context) ([]model.StatWidget, error) {
	if c.Querier == nil {
		return nil, nil
	}
	cost, err := c.Querier.GCPCost(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcp cost: %w", err)
	}
	return []model.StatWidget{{
		Key:   "cost_gcp_30d",
		Title: "GCP Cost",
		Value: fmt.Sprintf("%.2f", cost),
		Unit:  "AUD",
		Group: "Cost (30d)",
		Order: 40,
	}}, nil
}

// CostPerUserProvider supplies the inputs for the derived cost-per-active-user
// metric. It is satisfied by combining the usage and cost queriers.
type CostPerUserProvider struct {
	Cost  CostQuerier
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
		v, err := c.Provider.Cost.GCPCost(ctx)
		if err != nil {
			return nil, fmt.Errorf("gcp cost: %w", err)
		}
		gcp = v
	}

	total := gcp + c.Provider.Fixed.Supabase + c.Provider.Fixed.Redis + c.Provider.Fixed.Auth0
	perUser := total / float64(mau)

	return []model.StatWidget{{
		Key:   "cost_per_active_user_30d",
		Title: "Cost per Active User",
		Value: fmt.Sprintf("%.3f", perUser),
		Unit:  "AUD",
		Group: "Cost (30d)",
		Order: 44,
	}}, nil
}
