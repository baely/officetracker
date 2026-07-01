//go:build !bigquery

package stats

import "context"

// newBigQueryQuerier is the no-op stub used when the BigQuery adapter is not
// compiled in. It returns a nil querier so BuildRegistry skips the
// BigQuery-backed collectors. Build with `-tags bigquery` to use the real
// implementation in bigquery.go.
func newBigQueryQuerier(_ context.Context, _ Config) (bqQuerier, error) {
	return nil, nil
}
