package v1

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/baely/officetracker/pkg/model"
)

// exportEscaper keeps every CSV record on one physical line: notes can contain
// newlines, which would otherwise become (valid but hard to read) multi-line
// quoted fields.
var exportEscaper = strings.NewReplacer("\r\n", `\n`, "\n", `\n`, "\r", `\n`)

// ExportData returns a zip archive containing the user's data as CSV files.
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
		for _, row := range table.Rows {
			cells := make([]string, len(row))
			for j, cell := range row {
				cells[j] = exportEscaper.Replace(cell)
			}
			if err := cw.Write(cells); err != nil {
				return model.Response{}, fmt.Errorf("failed to write %s.csv: %w", table.Name, err)
			}
		}
		cw.Flush()
		if err := cw.Error(); err != nil {
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
