package report

import (
	"fmt"
	"path"
	"time"

	"github.com/google/uuid"

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
	months := getMonths(start, end)

	for _, month := range months {
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

func getMonths(start, end time.Time) []time.Time {
	var months []time.Time
	start = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location())

	for start.Before(end) {
		months = append(months, start)
		start = start.AddDate(0, 1, 0)
	}

	return months
}

// generateFilename generates a filename
func generateFilename(base string, ext string) string {
	id := uuid.NewString()
	filename := id + "." + ext
	return path.Join(base, filename)
}
