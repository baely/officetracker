package report

import (
	"fmt"
	"time"

	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/pkg/model"
)

type Key struct {
	Month time.Month
	Year  int
}

type Report struct {
	Months map[Key]model.MonthState
}

func (r Report) Get(month time.Month, year int) model.MonthState {
	key := Key{
		Month: month,
		Year:  year,
	}
	return r.Months[key]
}

type Reporter interface {
	Generate(userID int, start, end time.Time) (Report, error)
	GenerateCSV(userID int, start, end time.Time) ([]byte, error)
	GeneratePDF(userID int, name string, start, end time.Time) ([]byte, error)
}

type fileReporter struct {
	db database.Databaser
}

func New(db database.Databaser) Reporter {
	return &fileReporter{
		db: db,
	}
}

func (r *fileReporter) Generate(userID int, start, end time.Time) (Report, error) {
	report := Report{
		Months: make(map[Key]model.MonthState),
	}

	for month := range getMonths(start, end) {
		key := Key{
			Month: month.Month(),
			Year:  month.Year(),
		}
		monthData, err := r.db.GetMonth(userID, int(month.Month()), month.Year())
		if err != nil {
			err = fmt.Errorf("failed to get month state: %w", err)
			return Report{}, err
		}
		report.Months[key] = monthData
	}

	return report, nil
}

func getMonths(start, end time.Time) func(func(time.Time) bool) {
	start = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location())
	return func(yield func(time.Time) bool) {
		for {
			if !start.Before(end) {
				return
			}
			if !yield(start) {
				break
			}
			start = start.AddDate(0, 1, 0)
		}
	}
}

func getDays(start, end time.Time) func(func(time.Time) bool) {
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	return func(yield func(time.Time) bool) {
		if !start.Before(end) {
			return
		}
		if start.Weekday() > 0 && start.Weekday() < 6 {
			if !yield(start) {
				return
			}
		}
		start = start.AddDate(0, 0, 1)
	}
}
