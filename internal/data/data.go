package data

import (
	"bufio"
	"bytes"
	"fmt"
	"time"

	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/models"
	"github.com/baely/officetracker/internal/util"
)

var (
	melbourneLocation, _ = time.LoadLocation("Australia/Melbourne")
)

func GenerateSummary(db database.Databaser, userId string, month, year int) (models.Summary, error) {
	bankYear := year
	if month >= 10 {
		bankYear++
	}

	entries, err := db.GetEntriesForBankYear(userId, bankYear)
	if err != nil {
		return models.Summary{}, err
	}

	monthSet := make(map[string]bool)
	var monthData []*models.MonthSummary

	monthIter := -1

	var totalDays, totalPresent int
	for _, e := range entries {
		for _, state := range e.Days {
			if state < 1 || state > 2 {
				continue
			}

			entryMonth := fmt.Sprintf("%s %d", time.Month(e.Month).String(), e.Year)
			if _, ok := monthSet[entryMonth]; !ok {
				monthIter++
				monthSet[entryMonth] = true
				monthData = append(monthData, &models.MonthSummary{
					MonthUri:   fmt.Sprintf("/%d/%d", e.Year, e.Month),
					MonthLabel: entryMonth,
				})
			}

			data := monthData[monthIter]

			if state == util.Office {
				totalPresent++
				data.TotalPresent++

				totalDays++
				data.TotalDays++
			}
			if state == util.WFH {
				totalDays++
				data.TotalDays++
			}
		}
	}

	for _, data := range monthData {
		data.Percent = fmt.Sprintf("%.2f", float64(data.TotalPresent)/float64(data.TotalDays)*100)
	}

	return models.Summary{
		TotalDays:    totalDays,
		TotalPresent: totalPresent,
		Percent:      fmt.Sprintf("%.2f", float64(totalPresent)/float64(totalDays)*100),
		MonthData:    monthData,
	}, nil
}

func GenerateCsv(db database.Databaser, userId string) ([]byte, error) {
	entries, err := db.GetAllEntries(userId)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	w := bufio.NewWriter(&b)

	fmt.Fprintf(w, "%s,%s,%s\n", "Date", "Created Date", "Presence")

	for _, e := range entries {
		for day, state := range e.Days {
			explanation := util.StateToString(state)
			fmt.Fprintf(w, "%d-%d-%d,%s,%s\n", e.Year, e.Month, day, e.CreateDate.In(melbourneLocation).Format("2006-01-02 15:04:05"), explanation)
		}
	}

	w.Flush()

	return b.Bytes(), nil
}
