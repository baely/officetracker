package database

import (
	"database/sql"
	"encoding/json"

	"github.com/baely/officetracker/pkg/model"
)

// exportUserHeader is the header of the user summary table: it holds
// field/value rows rather than one row per database record.
var exportUserHeader = []string{"field", "value"}

// exportPreferenceFields names the user_preferences values in the order the
// export queries select them.
var exportPreferenceFields = []string{
	"theme", "weather_enabled", "time_based_enabled", "location",
	"schedule_monday_state", "schedule_tuesday_state", "schedule_wednesday_state",
	"schedule_thursday_state", "schedule_friday_state", "schedule_saturday_state",
	"schedule_sunday_state", "tracking_year_start_month", "target_percent",
}

// exportAccountRows renders one linked account as field/value rows: the Auth0
// subject followed by the recognisable profile fields that are present.
func exportAccountRows(sub, profileJSON string) [][]string {
	rows := [][]string{{"sub", sub}}
	var profile map[string]interface{}
	if err := json.Unmarshal([]byte(profileJSON), &profile); err != nil {
		return rows
	}
	for _, field := range []string{"nickname", "name", "email", "picture"} {
		if v, ok := profile[field].(string); ok && v != "" {
			rows = append(rows, []string{field, v})
		}
	}
	return rows
}

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
