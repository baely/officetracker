package v1

import (
	"fmt"
	"time"

	"github.com/baely/officetracker/pkg/model"
)

// GetReport godoc
//
//	@Summary		Download PDF attendance report
//	@Description	Generate and download attendance report as PDF for an academic year
//	@Tags			report
//	@Accept			json
//	@Produce		application/pdf
//	@Param			year	path		int		true	"Academic year"
//	@Param			name	query		string	false	"Name to include in report"
//	@Success		200		{file}		binary
//	@Failure		400		{object}	model.Error
//	@Failure		500		{object}	model.Error
//	@Security		BearerAuth
//	@Security		CookieAuth
//	@Router			/report/pdf/{year}-attendance [get]
func (i *Service) GetReport(req model.GetReportRequest) (model.Response, error) {
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

// GetReportCSV godoc
//
//	@Summary		Download CSV attendance report
//	@Description	Generate and download attendance report as CSV for an academic year
//	@Tags			report
//	@Accept			json
//	@Produce		text/csv
//	@Param			year	path	int	true	"Academic year"
//	@Success		200		{file}	binary
//	@Failure		400		{object}	model.Error
//	@Failure		500		{object}	model.Error
//	@Security		BearerAuth
//	@Security		CookieAuth
//	@Router			/report/csv/{year}-attendance [get]
func (i *Service) GetReportCSV(req model.GetReportCSVRequest) (model.Response, error) {
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
