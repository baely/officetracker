package data

import (
	"bufio"
	"bytes"
	"fmt"
	"time"

	"github.com/baely/officetracker/internal/database"
)

var (
	melbourneLocation, _ = time.LoadLocation("Australia/Melbourne")
)

func GenerateCsv(db *database.Client, userId string) ([]byte, error) {
	entries, err := db.GetEntries(userId)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	w := bufio.NewWriter(&b)

	fmt.Fprintf(w, "%s,%s,%s,%s\n", "Date", "Created Date", "Presence", "Reason")

	var rows []string
	lastDate := time.Time{}
	for _, e := range entries {
		s := fmt.Sprintf("%s,%s,%s,%s\n", e.Date.Format("2006-01-02"), e.CreatedDate.In(melbourneLocation).Format("2006-01-02 15:04:05"), e.Presence, e.Reason)
		if e.Date != lastDate {
			rows = append(rows, s)
			lastDate = e.Date
		} else {
			rows[len(rows)-1] = s
		}
	}
	for _, row := range rows {
		fmt.Fprintf(w, row)
	}

	w.Flush()

	return b.Bytes(), nil
}
