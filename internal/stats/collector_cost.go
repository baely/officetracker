package stats

import (
	"context"
	"fmt"

	"github.com/baely/officetracker/pkg/model"
)

// FixedCostConfig holds the fixed/known monthly cost for a platform whose
// billing is flat or free (Supabase, Redis Cloud, Auth0). These are configured
// constants rather than queried, because at current scale they do not vary.
type FixedCostConfig struct {
	Supabase float64
	Redis    float64
	Auth0    float64
}

// FixedCostCollector emits one cost widget per non-GCP platform using the
// configured constants.
type FixedCostCollector struct {
	Config FixedCostConfig
}

func (c FixedCostCollector) Name() string { return "fixed_cost" }

func (c FixedCostCollector) Collect(_ context.Context) ([]model.StatWidget, error) {
	platforms := []struct {
		key   string
		title string
		cost  float64
		order int
	}{
		{"cost_supabase_30d", "Supabase Cost", c.Config.Supabase, 41},
		{"cost_redis_30d", "Redis Cloud Cost", c.Config.Redis, 42},
		{"cost_auth0_30d", "Auth0 Cost", c.Config.Auth0, 43},
	}

	var widgets []model.StatWidget
	for _, p := range platforms {
		widgets = append(widgets, model.StatWidget{
			Key:    p.key,
			Title:  p.title,
			Value:  fmt.Sprintf("%.2f", p.cost),
			Prefix: "$",
			Group:  "Cost (30d)",
			Order:  p.order,
		})
	}
	return widgets, nil
}
