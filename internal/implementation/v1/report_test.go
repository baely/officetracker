package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/baely/officetracker/internal/report"
	"github.com/baely/officetracker/pkg/model"
	"github.com/baely/officetracker/testutil/mocks"
)

func TestGetReport(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockReporter := report.New(mockDB)
	service := New(mockDB, mockReporter)

	// Setup test data
	mockDB.SetDay(1, 4, 11, 2024, model.DayState{State: model.StateWorkFromOffice})
	schedulePrefs := model.SchedulePreferences{
		Monday: model.StateWorkFromOffice,
	}
	mockDB.SetSchedulePreferences(1, schedulePrefs)

	req := model.GetReportRequest{
		Meta: model.GetReportRequestMeta{
			UserID: 1,
			Year:   2024,
		},
		Name: "Test User",
	}

	resp, err := service.GetReport(req)

	require.NoError(t, err)
	assert.Equal(t, "application/pdf", resp.ContentType)
	assert.NotEmpty(t, resp.Data)

	// Verify it's a PDF
	pdfBytes, ok := resp.Data.([]byte)
	require.True(t, ok)
	assert.Equal(t, "%PDF-", string(pdfBytes[:5]))
}

func TestGetReport_ErrorHandling(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnGetMonth = assert.AnError
	mockReporter := report.New(mockDB)
	service := New(mockDB, mockReporter)

	req := model.GetReportRequest{
		Meta: model.GetReportRequestMeta{
			UserID: 1,
			Year:   2024,
		},
	}

	_, err := service.GetReport(req)
	assert.Error(t, err)
}

func TestGetReportCSV(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockReporter := report.New(mockDB)
	service := New(mockDB, mockReporter)

	// Setup test data for academic year 2024 (Oct 2023 - Sep 2024)
	// Use actual weekdays: Oct 2, 2023 is Monday, Oct 3, 2023 is Tuesday
	mockDB.SetDay(1, 2, 10, 2023, model.DayState{State: model.StateWorkFromOffice})
	mockDB.SetDay(1, 3, 10, 2023, model.DayState{State: model.StateWorkFromHome})
	schedulePrefs := model.SchedulePreferences{
		Monday:  model.StateWorkFromOffice,
		Tuesday: model.StateWorkFromHome,
	}
	mockDB.SetSchedulePreferences(1, schedulePrefs)

	req := model.GetReportCSVRequest{
		Meta: model.GetReportCSVRequestMeta{
			UserID: 1,
			Year:   2024,
		},
	}

	resp, err := service.GetReportCSV(req)

	require.NoError(t, err)
	assert.Equal(t, "text/csv", resp.ContentType)
	assert.NotEmpty(t, resp.Data)

	// Verify it's a CSV
	csvBytes, ok := resp.Data.([]byte)
	require.True(t, ok)
	csvString := string(csvBytes)
	assert.Contains(t, csvString, "Date,State")
	assert.Contains(t, csvString, "2023-10-02,Office") // Monday office
	assert.Contains(t, csvString, "2023-10-03,Home")   // Tuesday home
}

func TestGetReportCSV_ErrorHandling(t *testing.T) {
	mockDB := mocks.NewMockDB()
	mockDB.ErrorOnGetMonth = assert.AnError
	mockReporter := report.New(mockDB)
	service := New(mockDB, mockReporter)

	req := model.GetReportCSVRequest{
		Meta: model.GetReportCSVRequestMeta{
			UserID: 1,
			Year:   2024,
		},
	}

	_, err := service.GetReportCSV(req)
	assert.Error(t, err)
}
