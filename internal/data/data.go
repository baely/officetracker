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

type MonthSummary struct {
	MonthUri     string
	TotalDays    int
	TotalPresent int
	Percent      string
}

type Summary struct {
	TotalDays    int
	TotalPresent int
	Percent      string
	MonthData    map[string]*MonthSummary
}

func GenerateSummary(db *database.Client, userId string) (Summary, error) {
	entries, err := db.GetLatestEntries(userId)
	if err != nil {
		return Summary{}, err
	}

	monthData := make(map[string]*MonthSummary)

	var totalDays, totalPresent int
	for _, e := range entries {
		month := e.Date.Format("January 2006")
		if _, ok := monthData[month]; !ok {
			monthData[month] = &MonthSummary{
				MonthUri: fmt.Sprintf("/%s", e.Date.Format("2006-01")),
			}
		}
		data, _ := monthData[month]

		if e.Presence == "office" {
			totalPresent++
			data.TotalPresent++
		}
		totalDays++
		data.TotalDays++
	}

	for _, data := range monthData {
		data.Percent = fmt.Sprintf("%.2f", float64(data.TotalPresent)/float64(data.TotalDays)*100)
	}

	return Summary{
		TotalDays:    totalDays,
		TotalPresent: totalPresent,
		Percent:      fmt.Sprintf("%.2f", float64(totalPresent)/float64(totalDays)*100),
		MonthData:    monthData,
	}, nil
}

func GenerateCsv(db *database.Client, userId string) ([]byte, error) {
	entries, err := db.GetLatestEntries(userId)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	w := bufio.NewWriter(&b)

	fmt.Fprintf(w, "%s,%s,%s,%s\n", "Date", "Created Date", "Presence", "Reason")

	for _, e := range entries {
		fmt.Fprintf(w, "%s,%s,%s,%s\n", e.Date.Format("2006-01-02"), e.CreatedDate.In(melbourneLocation).Format("2006-01-02 15:04:05"), e.Presence, e.Reason)
	}

	w.Flush()

	return b.Bytes(), nil
}
