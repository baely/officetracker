package main

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/tests/blackbox/utils"
)

var (
	httpClient = http.DefaultClient
)

type Suite struct {
	cfg Config
	db  database.Databaser
}

func (s Suite) Run(t *testing.T) {
	for _, endpoint := range cases {
		t.Run(endpoint.Path, func(t *testing.T) {
			for _, tc := range endpoint.Cases {
				t.Run(tc.Name, func(t *testing.T) {
					// Seed data
					if tc.DataSeed != nil {
						require.NoError(t, tc.DataSeed(s.db))
					}

					req, err := utils.NewRequest(s.cfg.Host, tc.Method, endpoint.Path, tc.RequestPayload)
					require.NoError(t, err)

					resp, err := httpClient.Do(req)
					require.NoError(t, err)
					defer resp.Body.Close()

					// Check status code
					require.Equal(t, tc.ResponseStatus, resp.StatusCode)

					// Check response body
					if tc.ResponsePayload != "" {
						body, err := io.ReadAll(resp.Body)
						require.NoError(t, err)
						require.JSONEq(t, string(body), tc.ResponsePayload)
					}

					// Cleanup data
					if tc.DataCleanup != nil {
						require.NoError(t, tc.DataCleanup(s.db))
					}
				})
			}
		})
	}
}
