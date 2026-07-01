package stats

import "github.com/kelseyhightower/envconfig"

// Config controls the stats collection pipeline. All fields are optional; when
// BigQuery settings are absent, the usage/GCP-cost collectors are simply
// skipped and the dashboard shows whatever DB-backed and fixed-cost widgets are
// available.
type Config struct {
	// BigQuery configuration for usage + billing queries.
	BQProjectID    string `envconfig:"STATS_BQ_PROJECT_ID"`
	BQLogsTable    string `envconfig:"STATS_BQ_LOGS_TABLE"`    // fully-qualified: dataset.table
	BQBillingTable string `envconfig:"STATS_BQ_BILLING_TABLE"` // fully-qualified: dataset.table

	// Fixed monthly costs (AUD) for non-GCP platforms.
	CostSupabase float64 `envconfig:"STATS_COST_SUPABASE"`
	CostRedis    float64 `envconfig:"STATS_COST_REDIS"`
	CostAuth0    float64 `envconfig:"STATS_COST_AUTH0"`
}

// LoadConfig loads stats configuration from the environment.
func LoadConfig() (Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// FixedCosts returns the configured fixed-cost constants.
func (c Config) FixedCosts() FixedCostConfig {
	return FixedCostConfig{
		Supabase: c.CostSupabase,
		Redis:    c.CostRedis,
		Auth0:    c.CostAuth0,
	}
}

// BigQueryEnabled reports whether enough config is present to run BigQuery-backed
// collectors.
func (c Config) BigQueryEnabled() bool {
	return c.BQProjectID != "" && c.BQLogsTable != ""
}
