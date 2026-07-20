package v1

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"

	"github.com/baely/officetracker/pkg/model"
)

// ExportData returns a zip archive containing one CSV file per database table
// holding the user's data, each file named after its table.
func (i *Service) ExportData(req model.ExportDataRequest) (model.Response, error) {
	tables, err := i.db.ExportUserData(req.Meta.UserID)
	if err != nil {
		return model.Response{}, fmt.Errorf("failed to export user data: %w", err)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, table := range tables {
		f, err := zw.Create(table.Name + ".csv")
		if err != nil {
			return model.Response{}, fmt.Errorf("failed to create %s.csv: %w", table.Name, err)
		}
		cw := csv.NewWriter(f)
		if err := cw.Write(table.Header); err != nil {
			return model.Response{}, fmt.Errorf("failed to write %s.csv header: %w", table.Name, err)
		}
		// WriteAll flushes the writer.
		if err := cw.WriteAll(table.Rows); err != nil {
			return model.Response{}, fmt.Errorf("failed to write %s.csv: %w", table.Name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return model.Response{}, fmt.Errorf("failed to finalise export zip: %w", err)
	}

	return model.Response{
		ContentType: "application/zip",
		Data:        buf.Bytes(),
	}, nil
}
