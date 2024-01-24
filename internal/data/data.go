package data

import (
	"bufio"
	"bytes"
	"fmt"
	"time"

	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/util"
)

var (
	melbourneLocation, _ = time.LoadLocation("Australia/Melbourne")
)

type MonthSummary struct {
	MonthUri     string
	MonthLabel   string
	TotalDays    int
	TotalPresent int
	Percent      string
}

type Summary struct {
	TotalDays    int
	TotalPresent int
	Percent      string
	MonthData    []*MonthSummary
	MonthKeys    []string
}

func GenerateSummary(db *database.Client, userId string, month, year int) (Summary, error) {
	bankYear := year
	if month >= 10 {
		bankYear++
	}

	entries, err := db.GetEntriesForBankYear(userId, bankYear)
	if err != nil {
		return Summary{}, err
	}

	monthSet := make(map[string]bool)
	var monthData []*MonthSummary

	monthIter := -1

	var totalDays, totalPresent int
	for _, e := range entries {
		if e.State < 1 || e.State > 2 {
			continue
		}

		entryMonth := fmt.Sprintf("%s %d", time.Month(e.Month).String(), e.Year)
		if _, ok := monthSet[entryMonth]; !ok {
			monthIter++
			monthSet[entryMonth] = true
			monthData = append(monthData, &MonthSummary{
				MonthUri:   fmt.Sprintf("/%d/%d", e.Year, e.Month),
				MonthLabel: entryMonth,
			})
		}

		data := monthData[monthIter]

		if e.State == util.Office {
			totalPresent++
			data.TotalPresent++

			totalDays++
			data.TotalDays++
		}
		if e.State == util.WFH {
			totalDays++
			data.TotalDays++
		}
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

	fmt.Fprintf(w, "%s,%s,%s\n", "Date", "Created Date", "Presence")

	for _, e := range entries {
		explanation := util.StateToString(e.State)
		fmt.Fprintf(w, "%d-%d-%d,%s,%s\n", e.Year, e.Month, e.Day, e.CreateDate.In(melbourneLocation).Format("2006-01-02 15:04:05"), explanation)
	}

	w.Flush()

	return b.Bytes(), nil
}
