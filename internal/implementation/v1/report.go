package v1

import (
	"fmt"
	"time"

	"github.com/baely/officetracker/pkg/model"
)

func (i *implementation) GetReport(req model.GetReportRequest) (model.Response, error) {
	var start, end time.Time

	start = time.Date(req.Meta.Year-1, time.October, 1, 0, 0, 0, 0, time.Local)
	end = time.Date(req.Meta.Year, time.October, 1, 0, 0, 0, 0, time.Local)

	report, err := i.reporter.GeneratePDF(req.Meta.UserID, req.Name, start, end)
	if err != nil {
		err = fmt.Errorf("failed to generate pdf report: %w", err)
		return model.Response{}, err
	}

	return model.Response{
		ContentType: "application/pdf",
		Data:        report,
	}, nil
}

func (i *implementation) GetReportCSV(req model.GetReportCSVRequest) (model.Response, error) {
	var start, end time.Time

	start = time.Date(req.Meta.Year-1, time.October, 1, 0, 0, 0, 0, time.Local)
	end = time.Date(req.Meta.Year, time.October, 1, 0, 0, 0, 0, time.Local)

	report, err := i.reporter.GenerateCSV(req.Meta.UserID, start, end)
	if err != nil {
		err = fmt.Errorf("failed to generate csv report: %w", err)
		return model.Response{}, err
	}

	return model.Response{
		ContentType: "text/csv",
		Data:        report,
	}, nil
}
