package v1

import (
	"fmt"

	"github.com/baely/officetracker/pkg/model"
)

// GetStats returns the latest persisted stats snapshot for the public
// dashboard. It is unauthenticated and returns only aggregate, non-identifiable
// data.
func (i *Service) GetStats(_ model.GetStatsRequest) (model.GetStatsResponse, error) {
	widgets, computedAt, err := i.db.GetLatestStatsSnapshot()
	if err != nil {
		return model.GetStatsResponse{}, fmt.Errorf("failed to get stats snapshot: %w", err)
	}

	resp := model.GetStatsResponse{
		Widgets: widgets,
	}
	if !computedAt.IsZero() {
		resp.ComputedAt = computedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return resp, nil
}
