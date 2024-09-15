package report

import (
	"bytes"
	"fmt"
	"time"

	"github.com/baely/officetracker/pkg/model"
)

type csvLine struct {
	Date  string
	State string
}

func (r *fileReporter) GenerateCSV(userID int, start, end time.Time) ([]byte, error) {
	report, err := r.Generate(userID, start, end)
	if err != nil {
		err = fmt.Errorf("failed to generate report: %w", err)
		return nil, err
	}

	days := getDays(start, end)

	var lines []csvLine

	for _, day := range days {
		monthData := report.Get(day.Month(), day.Year())
		state := monthData.Days[day.Day()].State
		lines = append(lines, csvLine{
			Date:  day.Format("2006-01-02"),
			State: getState(state),
		})
	}

	return buildCsv(lines), nil
}

func getState(state model.State) string {
	switch state {
	case model.StateWorkFromHome:
		return "Home"
	case model.StateWorkFromOffice:
		return "Office"
	case model.StateOther:
		fallthrough
	case model.StateUntracked:
		fallthrough
	default:
		return ""
	}
}

func buildCsv(lines []csvLine) []byte {
	buf := new(bytes.Buffer)
	buf.WriteString("Date,State\n")
	for _, line := range lines {
		buf.WriteString(fmt.Sprintf("%s,%s\n", line.Date, line.State))
	}
	return buf.Bytes()
}
