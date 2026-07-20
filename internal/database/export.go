package database

import (
	"database/sql"

	"github.com/baely/officetracker/pkg/model"
)

// exportQuery describes one table's contribution to a user data export: the
// table name, the exported column headers, and the query producing exactly
// those columns. Queries must COALESCE nullable columns so every value scans
// into a string.
type exportQuery struct {
	name   string
	header []string
	query  string
}

// scanExportRows drains rows into string cells using database/sql's default
// conversions (ints, bools and timestamps all format losslessly).
func scanExportRows(rows *sql.Rows, table *model.ExportTable) error {
	defer rows.Close()
	for rows.Next() {
		values := make([]string, len(table.Header))
		ptrs := make([]interface{}, len(values))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		table.Rows = append(table.Rows, values)
	}
	return rows.Err()
}
