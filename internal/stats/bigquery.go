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

func (q *bigQueryQuerier) queryScalar(ctx context.Context, sql string) (float64, error) {
	job, err := q.client.Query(sql).Run(ctx)
	if err != nil {
		return 0, err
	}
	it, err := job.Read(ctx)
	if err != nil {
		return 0, err
	}
	var row []bigquery.Value
	err = it.Next(&row)
	if err == iterator.Done || len(row) == 0 || row[0] == nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	switch v := row[0].(type) {
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	default:
		return 0, fmt.Errorf("unexpected scalar type %T", v)
	}
}

func (q *bigQueryQuerier) MAU(ctx context.Context) (int, error) {
	sql := fmt.Sprintf(`
SELECT COUNT(DISTINCT CAST(jsonPayload.userID AS INT64)) AS v
FROM `+"`%s`"+`
WHERE jsonPayload.message = 'request processed'
  AND SAFE_CAST(jsonPayload.userID AS INT64) > 0
  AND timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY)`, q.logsTable)
	v, err := q.queryScalar(ctx, sql)
	return int(v), err
}

func (q *bigQueryQuerier) AvgDAU(ctx context.Context) (float64, error) {
	sql := fmt.Sprintf(`
SELECT AVG(daily) AS v FROM (
  SELECT DATE(timestamp) d, COUNT(DISTINCT CAST(jsonPayload.userID AS INT64)) daily
  FROM `+"`%s`"+`
  WHERE jsonPayload.message = 'request processed'
    AND SAFE_CAST(jsonPayload.userID AS INT64) > 0
    AND timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY)
  GROUP BY d)`, q.logsTable)
	return q.queryScalar(ctx, sql)
}

func (q *bigQueryQuerier) RequestCount(ctx context.Context) (int, error) {
	// Authenticated requests only: a valid userID excludes crawler/bot traffic,
	// which cannot hold a session.
	sql := fmt.Sprintf(`
SELECT COUNT(*) AS v
FROM `+"`%s`"+`
WHERE jsonPayload.message = 'request processed'
  AND SAFE_CAST(jsonPayload.userID AS INT64) > 0
  AND timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY)`, q.logsTable)
	v, err := q.queryScalar(ctx, sql)
	return int(v), err
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
	sql := fmt.Sprintf(`
SELECT SUM(cost) AS v
FROM `+"`%s`"+`
WHERE project.id = @projectID
  AND usage_start_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY)`, q.billingTable)

	query := q.client.Query(sql)
	query.Parameters = []bigquery.QueryParameter{
		{Name: "projectID", Value: q.billingProjectID},
	}
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
	if err == iterator.Done || len(row) == 0 || row[0] == nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	switch v := row[0].(type) {
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	default:
		return 0, fmt.Errorf("unexpected scalar type %T", v)
	}
}
