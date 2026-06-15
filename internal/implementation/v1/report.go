package v1

import (
	"fmt"

	"github.com/baely/officetracker/internal/util"
	"github.com/baely/officetracker/pkg/model"
)

func (i *Service) GetReport(req model.GetReportRequest) (model.Response, error) {
	startMonth, err := i.trackingStartMonth(req.Meta.UserID)
	if err != nil {
		return model.Response{}, err
	}

	start, end := util.TrackingYearRange(req.Meta.Year, startMonth)

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

func (i *Service) GetReportCSV(req model.GetReportCSVRequest) (model.Response, error) {
	startMonth, err := i.trackingStartMonth(req.Meta.UserID)
	if err != nil {
		return model.Response{}, err
	}

	start, end := util.TrackingYearRange(req.Meta.Year, startMonth)

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
