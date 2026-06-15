package util

import "time"

// Tracking-year helpers.
//
// Entries are stored against their raw calendar (year, month). A "tracking year"
// is a presentation grouping of 12 consecutive months starting at a configurable
// start month (1-12, e.g. 10 = October). It lets users align attendance reporting
// to whatever 12-month cycle their workplace uses.
//
// A tracking year is labelled by the calendar year in which it *ends*
// (e.g. an October start means the period Oct 2023 - Sep 2024 is labelled 2024).
// When the start month is January the tracking year coincides with the calendar
// year, so it is labelled by that same year.

// NormaliseStartMonth clamps an arbitrary value to a valid month (1-12),
// falling back to the October default when out of range.
func NormaliseStartMonth(startMonth int) int {
	if startMonth < 1 || startMonth > 12 {
		return 10
	}
	return startMonth
}

// TrackingYear returns the tracking-year label that a given calendar month
// (1-12) in calendar year cy belongs to.
func TrackingYear(month, cy, startMonth int) int {
	startMonth = NormaliseStartMonth(startMonth)
	if startMonth == 1 {
		return cy
	}
	if month >= startMonth {
		return cy + 1
	}
	return cy
}

// TrackingYearCalendarYears returns the two calendar years spanned by the
// tracking year labelled ty: months >= startMonth belong to firstYear, months <
// startMonth belong to secondYear. For a January start both are equal to ty.
func TrackingYearCalendarYears(ty, startMonth int) (firstYear, secondYear int) {
	startMonth = NormaliseStartMonth(startMonth)
	if startMonth == 1 {
		return ty, ty
	}
	return ty - 1, ty
}

// TrackingYearRange returns the half-open [start, end) date range covering the
// tracking year labelled ty.
func TrackingYearRange(ty, startMonth int) (start, end time.Time) {
	startMonth = NormaliseStartMonth(startMonth)
	firstYear, _ := TrackingYearCalendarYears(ty, startMonth)
	start = time.Date(firstYear, time.Month(startMonth), 1, 0, 0, 0, 0, time.Local)
	end = start.AddDate(1, 0, 0)
	return start, end
}
