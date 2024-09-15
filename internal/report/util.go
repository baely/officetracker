package report

import (
	"path"
	"time"

	"github.com/google/uuid"

	"github.com/baely/officetracker/internal/database"
)

type ReportMonth struct {
	Month time.Month
	Year  int
}

type Report struct {
	Months []ReportMonth
}

type Reporter interface {
	Generate(userID int, start, end time.Time) (string, error)
	GenerateCSV(userID int, start, end time.Time) ([]byte, error)
	GeneratePDF(userID int, start, end time.Time) ([]byte, error)
}

type fileReporter struct {
	db database.Databaser
}

func New(db database.Databaser) Reporter {
	return &fileReporter{
		db: db,
	}
}

func (r *fileReporter) Generate(userID int, start, end time.Time) (string, error) {
	return "", nil
}

// generateFilename generates a filename
func generateFilename(base string, ext string) string {
	id := uuid.NewString()
	filename := id + "." + ext
	return path.Join(base, filename)
}
