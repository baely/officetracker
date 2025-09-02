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

	// Fetch schedule preferences
	schedulePrefs, err := r.db.GetSchedulePreferences(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule preferences: %w", err)
	}

	var lines []csvLine

	for day := range getDays(start, end) {
		monthData := report.Get(day.Month(), day.Year())
		dayState, exists := monthData.Days[day.Day()]
		
		var state model.State
		if exists {
			state = dayState.State
		} else {
			state = model.StateUntracked
		}

		// Check if this is a scheduled day that's untracked
		stateString := getState(state)
		if state == model.StateUntracked && isScheduledDay(day, schedulePrefs) {
			stateString = "Scheduled"
		}

		lines = append(lines, csvLine{
			Date:  day.Format("2006-01-02"),
			State: stateString,
		})
	}

	return buildCsv(lines), nil
}

func isScheduledDay(day time.Time, schedulePrefs model.SchedulePreferences) bool {
	switch day.Weekday() {
	case time.Sunday:
		return schedulePrefs.Sunday != model.StateUntracked
	case time.Monday:
		return schedulePrefs.Monday != model.StateUntracked
	case time.Tuesday:
		return schedulePrefs.Tuesday != model.StateUntracked
	case time.Wednesday:
		return schedulePrefs.Wednesday != model.StateUntracked
	case time.Thursday:
		return schedulePrefs.Thursday != model.StateUntracked
	case time.Friday:
		return schedulePrefs.Friday != model.StateUntracked
	case time.Saturday:
		return schedulePrefs.Saturday != model.StateUntracked
	default:
		return false
	}
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
