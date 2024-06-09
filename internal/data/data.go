package data

import (
	"time"
)

var (
	melbourneLocation, _ = time.LoadLocation("Australia/Melbourne")
)

//func GenerateSummary(db database.Databaser, userId string, month, year int) (models.Summary, error) {
//	bankYear := year
//	if month >= 10 {
//		bankYear++
//	}
//
//	entries, err := db.GetEntriesForBankYear(userId, bankYear)
//	if err != nil {
//		return models.Summary{}, err
//	}
//
//	monthSet := make(map[string]bool)
//	var monthData []*models.MonthSummary
//
//	monthIter := -1
//
//	var totalDays, totalPresent int
//	for _, e := range entries {
//		for _, state := range e.Days {
//			if state < 1 || state > 2 {
//				continue
//			}
//
//			entryMonth := fmt.Sprintf("%s %d", time.Month(e.Month).String(), e.Year)
//			if _, ok := monthSet[entryMonth]; !ok {
//				monthIter++
//				monthSet[entryMonth] = true
//				monthData = append(monthData, &models.MonthSummary{
//					MonthUri:   fmt.Sprintf("/%d/%d", e.Year, e.Month),
//					MonthLabel: entryMonth,
//				})
//			}
//
//			data := monthData[monthIter]
//
//			if state == util.Office {
//				totalPresent++
//				data.TotalPresent++
//
//				totalDays++
//				data.TotalDays++
//			}
//			if state == util.WFH {
//				totalDays++
//				data.TotalDays++
//			}
//		}
//	}
//
//	for _, data := range monthData {
//		data.Percent = fmt.Sprintf("%.2f", float64(data.TotalPresent)/float64(data.TotalDays)*100)
//	}
//
//	return models.Summary{
//		TotalDays:    totalDays,
//		TotalPresent: totalPresent,
//		Percent:      fmt.Sprintf("%.2f", float64(totalPresent)/float64(totalDays)*100),
//		MonthData:    monthData,
//	}, nil
//}
