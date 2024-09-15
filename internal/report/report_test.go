package report

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/baely/officetracker/internal/util/testutils"
	"github.com/baely/officetracker/pkg/model"
)

func TestFileReporter_GeneratePDF(t *testing.T) {
	db := &mockDatabaser{}
	reporter := New(db)

	var start, end time.Time
	start = time.Date(time.Now().Year(), 1, 1, 0, 0, 0, 0, time.Local)
	end = time.Date(time.Now().Year()+1, 1, 1, 0, 0, 0, 0, time.Local)

	report, err := reporter.GeneratePDF(1, "Bailey Butler", start, end)

	require.NoError(t, err)

	// save bytes to temp
	os.WriteFile("report.pdf", report, 0644)
}

type mockDatabaser struct {
	testutils.UnimplementedDatabaser
}

func (m *mockDatabaser) GetMonth(userID, month, year int) (model.MonthState, error) {
	return model.MonthState{}, nil
}
