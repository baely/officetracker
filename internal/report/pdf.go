package report

import "time"

func (r *fileReporter) GeneratePDF(userID int, start, end time.Time) ([]byte, error) {
	_, err := r.Generate(userID, start, end)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
