//go:build bigquery

// Package stats BigQuery adapter. Compiled only with `-tags bigquery` so the
// default server/standalone builds don't pull in the GCP SDK. Wire this up for
// the collector Cloud Run Job.
//
// Requires: go get cloud.google.com/go/bigquery
package stats

import (
	"context"
	"fmt"
	"math/big"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

type bigQueryQuerier struct {
	client           *bigquery.Client
	logsTable        string
	billingTable     string
	billingProjectID string
}

func newBigQueryQuerier(ctx context.Context, cfg Config) (bqQuerier, error) {
	client, err := bigquery.NewClient(ctx, cfg.BQProjectID)
	if err != nil {
		return nil, fmt.Errorf("bigquery client: %w", err)
	}
	return &bigQueryQuerier{
		client:           client,
		logsTable:        cfg.BQLogsTable,
		billingTable:     cfg.BQBillingTable,
		billingProjectID: cfg.BQBillingProjectID,
	}, nil
}

// toFloat64 coerces a BigQuery scalar to float64. It accepts every numeric type
// the driver may hand back - including *big.Rat for NUMERIC/BIGNUMERIC columns -
// and treats NULL as 0.
func toFloat64(v bigquery.Value) (float64, error) {
	switch x := v.(type) {
	case nil:
		return 0, nil
	case int64:
		return float64(x), nil
	case float64:
		return x, nil
	case *big.Rat:
		f, _ := x.Float64()
		return f, nil
	default:
		return 0, fmt.Errorf("unexpected scalar type %T", v)
	}
}

// queryScalar runs a query expected to yield a single scalar in the first column
// of the first row, returning 0 for no rows or NULL. Optional query parameters
// are bound when provided.
func (q *bigQueryQuerier) queryScalar(ctx context.Context, sql string, params ...bigquery.QueryParameter) (float64, error) {
	query := q.client.Query(sql)
	query.Parameters = params
	job, err := query.Run(ctx)
	if err != nil {
		return 0, err
	}
	it, err := job.Read(ctx)
	if err != nil {
		return 0, err
	}
	var row []bigquery.Value
	err = it.Next(&row)
	if err == iterator.Done || len(row) == 0 {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return toFloat64(row[0])
}

// Usage returns MAU, average DAU and request count over the trailing 30 days
// from a single scan of the request log. Only authenticated traffic (userID > 0)
// is counted, so crawler/bot requests are excluded. AvgDAU divides the total
// distinct user-days by a fixed 30-day denominator (zero-activity days count as
// 0), rather than averaging over active days only, which would overstate it.
func (q *bigQueryQuerier) Usage(ctx context.Context) (UsageStats, error) {
	sql := fmt.Sprintf(`
SELECT
  COUNT(DISTINCT uid) AS mau,
  COUNT(*) AS requests,
  COUNT(DISTINCT CONCAT(CAST(d AS STRING), '|', CAST(uid AS STRING))) / 30.0 AS avg_dau
FROM (
  SELECT DATE(timestamp, 'Australia/Melbourne') AS d,
         SAFE_CAST(jsonPayload.userID AS INT64) AS uid
  FROM `+"`%s`"+`
  WHERE jsonPayload.msg = 'request processed'
    AND SAFE_CAST(jsonPayload.userID AS INT64) > 0
    AND timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY)
)`, q.logsTable)

	job, err := q.client.Query(sql).Run(ctx)
	if err != nil {
		return UsageStats{}, err
	}
	it, err := job.Read(ctx)
	if err != nil {
		return UsageStats{}, err
	}
	var row []bigquery.Value
	err = it.Next(&row)
	if err == iterator.Done || len(row) < 3 {
		return UsageStats{}, nil
	}
	if err != nil {
		return UsageStats{}, err
	}
	mau, err := toFloat64(row[0])
	if err != nil {
		return UsageStats{}, err
	}
	reqs, err := toFloat64(row[1])
	if err != nil {
		return UsageStats{}, err
	}
	avgDAU, err := toFloat64(row[2])
	if err != nil {
		return UsageStats{}, err
	}
	return UsageStats{MAU: int(mau), Requests: int(reqs), AvgDAU: avgDAU}, nil
}

func (q *bigQueryQuerier) GCPCost(ctx context.Context) (float64, error) {
	if q.billingTable == "" {
		return 0, nil
	}
	// The Cloud Billing export table contains rows for every project under the
	// billing account, so we must scope to Office Tracker's project. Without a
	// configured project ID the cost would incorrectly include other projects,
	// so we refuse to report a misleading figure.
	if q.billingProjectID == "" {
		return 0, fmt.Errorf("STATS_BQ_BILLING_PROJECT_ID is required to scope billing to a single project")
	}
	// Net cost = list cost plus credits (free-tier/promotional credits are
	// stored as negative amounts in the repeated `credits` field, so summing
	// only `cost` would overstate the real bill). SAFE_CAST guards against the
	// billing export's `cost` column being NUMERIC rather than FLOAT64.
	sql := fmt.Sprintf(`
SELECT SAFE_CAST(SUM(cost) + SUM(IFNULL((SELECT SUM(c.amount) FROM UNNEST(credits) c), 0)) AS FLOAT64) AS v
FROM `+"`%s`"+`
WHERE project.id = @projectID
  AND usage_start_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY)`, q.billingTable)

	return q.queryScalar(ctx, sql, bigquery.QueryParameter{Name: "projectID", Value: q.billingProjectID})
}
