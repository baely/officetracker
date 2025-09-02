package report

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"

	"github.com/baely/officetracker/pkg/model"
)

// MonthlySummary represents attendance summary for a month
type MonthlySummary struct {
	Present int
	Total   int
	Percent float64
}

// PDF represents a PDF document with report data
type PDF struct {
	*gofpdf.Fpdf
	report             Report
	monthlySummaries   map[time.Time]MonthlySummary
	schedulePreferences model.SchedulePreferences
	name               string
	start, end         time.Time
}

// GeneratePDF creates a PDF report for the given user and time range
func (r *fileReporter) GeneratePDF(userID int, name string, start, end time.Time) ([]byte, error) {
	report, err := r.Generate(userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to generate report: %w", err)
	}

	// Fetch schedule preferences
	schedulePrefs, err := r.db.GetSchedulePreferences(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule preferences: %w", err)
	}

	p := newPDF(report, schedulePrefs, name, start, end)
	p.addCoverPage()

	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	if err := p.Output(bw); err != nil {
		return nil, fmt.Errorf("failed to write PDF to buffer: %w", err)
	}

	if err := bw.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush buffer: %w", err)
	}

	return buf.Bytes(), nil
}

func newPDF(report Report, schedulePrefs model.SchedulePreferences, name string, start, end time.Time) *PDF {
	f := gofpdf.New("P", "mm", "A4", "")
	f.SetMargins(15, 30, 15)
	f.AliasNbPages("{pages}")
	f.SetFooterFunc(func() {
		f.SetY(-15)
		f.SetFont("Arial", "I", 8)
		nameStr := ""
		if name != "" {
			nameStr = name + " - "
		}
		f.CellFormat(0, 10, fmt.Sprintf("Officetracker Attendance Report - %sPage %d of {pages}", nameStr, f.PageNo()), "", 0, "C", false, 0, "")
	})

	p := &PDF{
		Fpdf:               f,
		report:             report,
		schedulePreferences: schedulePrefs,
		name:               name,
		start:              start,
		end:                end,
	}
	p.generateSummaries()
	return p
}

func (p *PDF) addCoverPage() {
	p.AddPage()

	p.SetFont("Arial", "B", 28)
	p.Cell(40, 10, "Officetracker Attendance Report")
	p.Ln(15)

	nameStr := ""
	if p.name != "" {
		nameStr = p.name + " - "
	}

	p.SetFont("Arial", "I", 24)
	p.Cell(40, 10, fmt.Sprintf("%sBank Financial Year %d", nameStr, p.end.Year()))
	p.Ln(30)

	p.addSummaryTable()

	p.SetY(-45)
	p.SetFont("Arial", "I", 8)
	p.Cell(0, 10, "Disclaimer: This report is based on self-reported data submitted through https://officetracker.com.au and has been automatically generated.")

	for month := range getMonths(p.start, p.end) {
		p.addMonthPage(month)
	}
}

func (p *PDF) addSummaryTable() {
	p.SetFont("Arial", "B", 16)
	p.Cell(40, 10, "Summary Stats")
	p.Ln(15)

	p.SetFont("Arial", "B", 12)
	p.CellFormat(60, 10, padString("Month", 2, 2), "1", 0, "L", false, 0, "")
	p.CellFormat(40, 10, padString("Present", 2, 2), "1", 0, "L", false, 0, "")
	p.CellFormat(40, 10, padString("Total", 2, 2), "1", 0, "L", false, 0, "")
	p.CellFormat(40, 10, padString("Percent", 2, 2), "1", 0, "L", false, 0, "")
	p.Ln(10)

	var present, total int

	p.SetFont("Arial", "", 12)
	for month := range getMonths(p.start, p.end) {
		summary := p.monthlySummaries[month]
		present += summary.Present
		total += summary.Total
		p.CellFormat(60, 10, padString(month.Format("January 2006"), 2, 2), "1", 0, "L", false, 0, "")
		p.CellFormat(40, 10, padString(fmt.Sprintf("%d", summary.Present), 2, 2), "1", 0, "L", false, 0, "")
		p.CellFormat(40, 10, padString(fmt.Sprintf("%d", summary.Total), 2, 2), "1", 0, "L", false, 0, "")
		p.CellFormat(40, 10, padString(fmt.Sprintf("%.2f%%", summary.Percent), 2, 2), "1", 0, "L", false, 0, "")
		p.Ln(10)
	}

	percent := 0.0
	if total > 0 {
		percent = float64(present) / float64(total) * 100
	}

	p.SetFont("Arial", "B", 12)
	p.CellFormat(60, 10, padString("Total", 2, 2), "1", 0, "L", false, 0, "")
	p.CellFormat(40, 10, padString(fmt.Sprintf("%d", present), 2, 2), "1", 0, "L", false, 0, "")
	p.CellFormat(40, 10, padString(fmt.Sprintf("%d", total), 2, 2), "1", 0, "L", false, 0, "")
	p.CellFormat(40, 10, padString(fmt.Sprintf("%.2f%%", percent), 2, 2), "1", 0, "L", false, 0, "")
}

func (p *PDF) addMonthPage(month time.Time) {
	p.AddPage()

	p.SetFont("Arial", "B", 24)
	p.Cell(40, 10, month.Format("January 2006"))
	p.Ln(20)

	p.addMonthSummary(month)
	p.addMonthTable(month)
}

func (p *PDF) addMonthSummary(month time.Time) {
	p.SetFont("Arial", "B", 16)
	p.Cell(40, 10, "Summary Stats")
	p.Ln(15)

	p.SetFont("Arial", "B", 12)
	headers := []string{"Present", "Total", "Percent"}
	for _, header := range headers {
		p.CellFormat(40, 10, padString(header, 2, 2), "1", 0, "L", false, 0, "")
	}
	p.Ln(10)

	summary := p.monthlySummaries[month]
	p.CellFormat(40, 10, padString(fmt.Sprintf("%d", summary.Present), 2, 2), "1", 0, "L", false, 0, "")
	p.CellFormat(40, 10, padString(fmt.Sprintf("%d", summary.Total), 2, 2), "1", 0, "L", false, 0, "")
	p.CellFormat(40, 10, padString(fmt.Sprintf("%.2f%%", summary.Percent), 2, 2), "1", 0, "L", false, 0, "")
	p.Ln(15)
}

func (p *PDF) addMonthTable(month time.Time) {
	p.SetFont("Arial", "B", 16)
	p.Cell(40, 10, "Attendance Record")
	p.Ln(15)

	p.SetFont("Arial", "B", 10)
	headers := []string{"Date", "Day of Week", "Status"}
	for _, header := range headers {
		p.CellFormat(40, 8, padString(header, 2, 2), "1", 0, "L", false, 0, "")
	}
	p.Ln(8)

	p.SetFont("Arial", "", 10)
	for day := range getDays(month, month.AddDate(0, 1, 0)) {
		status := p.report.Get(day.Month(), day.Year()).Days[day.Day()].State
		statusStr := getStatusString(status)

		p.CellFormat(40, 6, padString(day.Format("02 January"), 2, 2), "1", 0, "L", false, 0, "")
		p.CellFormat(40, 6, padString(day.Format("Monday"), 2, 2), "1", 0, "L", false, 0, "")
		p.CellFormat(40, 6, padString(statusStr, 2, 2), "1", 0, "L", false, 0, "")
		p.Ln(6)
	}
}

func (p *PDF) generateSummaries() {
	p.monthlySummaries = make(map[time.Time]MonthlySummary)
	for month := range getMonths(p.start, p.end) {
		summary := MonthlySummary{}

		for _, state := range p.report.Get(month.Month(), month.Year()).Days {
			if state.State == model.StateWorkFromOffice {
				summary.Present++
				summary.Total++
			} else if state.State == model.StateWorkFromHome {
				summary.Total++
			}
		}

		// Count scheduled days that are untracked as expected office days
		scheduledDays := p.countScheduledDays(month.Year(), month.Month())
		summary.Total += scheduledDays

		if summary.Total > 0 {
			summary.Percent = float64(summary.Present) / float64(summary.Total) * 100
		}

		p.monthlySummaries[month] = summary
	}
}

func (p *PDF) countScheduledDays(year int, month time.Month) int {
	scheduledCount := 0
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	monthState := p.report.Get(month, year)

	for day := 1; day <= daysInMonth; day++ {
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		dayOfWeek := date.Weekday()
		
		// Check if this day is scheduled
		var isScheduled bool
		switch dayOfWeek {
		case time.Sunday:
			isScheduled = p.schedulePreferences.Sunday != model.StateUntracked
		case time.Monday:
			isScheduled = p.schedulePreferences.Monday != model.StateUntracked
		case time.Tuesday:
			isScheduled = p.schedulePreferences.Tuesday != model.StateUntracked
		case time.Wednesday:
			isScheduled = p.schedulePreferences.Wednesday != model.StateUntracked
		case time.Thursday:
			isScheduled = p.schedulePreferences.Thursday != model.StateUntracked
		case time.Friday:
			isScheduled = p.schedulePreferences.Friday != model.StateUntracked
		case time.Saturday:
			isScheduled = p.schedulePreferences.Saturday != model.StateUntracked
		}

		// Check if this day is untracked and scheduled
		if isScheduled {
			dayState, exists := monthState.Days[day]
			if !exists || dayState.State == model.StateUntracked {
				scheduledCount++
			}
		}
	}

	return scheduledCount
}

func getStatusString(status model.State) string {
	switch status {
	case model.StateWorkFromHome:
		return "Home"
	case model.StateWorkFromOffice:
		return "Office"
	case model.StateOther, model.StateUntracked:
		fallthrough
	default:
		return ""
	}
}

func padString(s string, paddingLeft, paddingRight int) string {
	return strings.Repeat(" ", paddingLeft) + s + strings.Repeat(" ", paddingRight)
}
