package fixtures

import (
	"time"

	"github.com/baely/officetracker/pkg/model"
)

// YearStateBuilder helps build complex YearState test data
type YearStateBuilder struct {
	year   int
	months map[int]*MonthStateBuilder
}

// NewYearState creates a new YearStateBuilder for the specified year
func NewYearState(year int) *YearStateBuilder {
	return &YearStateBuilder{
		year:   year,
		months: make(map[int]*MonthStateBuilder),
	}
}

// WithDay adds a day with the specified state
func (b *YearStateBuilder) WithDay(month, day int, state model.State) *YearStateBuilder {
	if _, ok := b.months[month]; !ok {
		b.months[month] = &MonthStateBuilder{days: make(map[int]model.DayState)}
	}
	b.months[month].days[day] = model.DayState{State: state}
	return b
}

// WithDays adds multiple days with the same state
func (b *YearStateBuilder) WithDays(month int, days []int, state model.State) *YearStateBuilder {
	if _, ok := b.months[month]; !ok {
		b.months[month] = &MonthStateBuilder{days: make(map[int]model.DayState)}
	}
	for _, day := range days {
		b.months[month].days[day] = model.DayState{State: state}
	}
	return b
}

// WithMonth adds an entire month of data
func (b *YearStateBuilder) WithMonth(month int, monthState model.MonthState) *YearStateBuilder {
	b.months[month] = &MonthStateBuilder{days: monthState.Days}
	return b
}

// Build constructs the final YearState
func (b *YearStateBuilder) Build() model.YearState {
	yearState := model.YearState{Months: make(map[int]model.MonthState)}
	for month, mb := range b.months {
		yearState.Months[month] = mb.Build()
	}
	return yearState
}

// MonthStateBuilder helps build MonthState test data
type MonthStateBuilder struct {
	days map[int]model.DayState
}

// NewMonthState creates a new MonthStateBuilder
func NewMonthState() *MonthStateBuilder {
	return &MonthStateBuilder{
		days: make(map[int]model.DayState),
	}
}

// WithDay adds a day with the specified state
func (b *MonthStateBuilder) WithDay(day int, state model.State) *MonthStateBuilder {
	b.days[day] = model.DayState{State: state}
	return b
}

// WithDays adds multiple days with the same state
func (b *MonthStateBuilder) WithDays(days []int, state model.State) *MonthStateBuilder {
	for _, day := range days {
		b.days[day] = model.DayState{State: state}
	}
	return b
}

// Build constructs the final MonthState
func (b *MonthStateBuilder) Build() model.MonthState {
	return model.MonthState{Days: b.days}
}

// SchedulePreferencesBuilder helps build SchedulePreferences test data
type SchedulePreferencesBuilder struct {
	prefs model.SchedulePreferences
}

// NewSchedulePreferences creates a new SchedulePreferencesBuilder
func NewSchedulePreferences() *SchedulePreferencesBuilder {
	return &SchedulePreferencesBuilder{
		prefs: model.SchedulePreferences{},
	}
}

// WithWeekday sets the state for a specific weekday
func (b *SchedulePreferencesBuilder) WithWeekday(day time.Weekday, state model.State) *SchedulePreferencesBuilder {
	switch day {
	case time.Sunday:
		b.prefs.Sunday = state
	case time.Monday:
		b.prefs.Monday = state
	case time.Tuesday:
		b.prefs.Tuesday = state
	case time.Wednesday:
		b.prefs.Wednesday = state
	case time.Thursday:
		b.prefs.Thursday = state
	case time.Friday:
		b.prefs.Friday = state
	case time.Saturday:
		b.prefs.Saturday = state
	}
	return b
}

// WithWorkdaysWFH sets all weekdays (Mon-Fri) to work from home
func (b *SchedulePreferencesBuilder) WithWorkdaysWFH() *SchedulePreferencesBuilder {
	b.prefs.Monday = model.StateWorkFromHome
	b.prefs.Tuesday = model.StateWorkFromHome
	b.prefs.Wednesday = model.StateWorkFromHome
	b.prefs.Thursday = model.StateWorkFromHome
	b.prefs.Friday = model.StateWorkFromHome
	return b
}

// WithWorkdaysOffice sets all weekdays (Mon-Fri) to work from office
func (b *SchedulePreferencesBuilder) WithWorkdaysOffice() *SchedulePreferencesBuilder {
	b.prefs.Monday = model.StateWorkFromOffice
	b.prefs.Tuesday = model.StateWorkFromOffice
	b.prefs.Wednesday = model.StateWorkFromOffice
	b.prefs.Thursday = model.StateWorkFromOffice
	b.prefs.Friday = model.StateWorkFromOffice
	return b
}

// WithHybrid sets a typical hybrid schedule (Mon/Wed/Fri office, Tue/Thu home)
func (b *SchedulePreferencesBuilder) WithHybrid() *SchedulePreferencesBuilder {
	b.prefs.Monday = model.StateWorkFromOffice
	b.prefs.Tuesday = model.StateWorkFromHome
	b.prefs.Wednesday = model.StateWorkFromOffice
	b.prefs.Thursday = model.StateWorkFromHome
	b.prefs.Friday = model.StateWorkFromOffice
	return b
}

// Build constructs the final SchedulePreferences
func (b *SchedulePreferencesBuilder) Build() model.SchedulePreferences {
	return b.prefs
}

// Date helpers for academic year boundaries

// AcademicYearStart returns October 1 of year-1 (start of academic year)
func AcademicYearStart(year int) time.Time {
	return time.Date(year-1, 10, 1, 0, 0, 0, 0, time.UTC)
}

// AcademicYearEnd returns September 30 of year (end of academic year)
func AcademicYearEnd(year int) time.Time {
	return time.Date(year, 9, 30, 0, 0, 0, 0, time.UTC)
}

// MonthBoundaries returns the first and last day of a month
func MonthBoundaries(year int, month int) (first, last time.Time) {
	first = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	// Last day is the 0th day of the next month
	last = time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC)
	return
}

// IsLeapYear returns true if the year is a leap year
func IsLeapYear(year int) bool {
	// Leap year if divisible by 4, except century years unless divisible by 400
	return (year%4 == 0 && year%100 != 0) || (year%400 == 0)
}

// DaysInMonth returns the number of days in a month
func DaysInMonth(year int, month int) int {
	return time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC).Day()
}

// LeapYearDates returns test dates for leap year testing
func LeapYearDates() []time.Time {
	return []time.Time{
		time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC), // Leap year
		time.Date(2023, 2, 28, 0, 0, 0, 0, time.UTC), // Non-leap
		time.Date(2000, 2, 29, 0, 0, 0, 0, time.UTC), // Century leap (divisible by 400)
		time.Date(1900, 2, 28, 0, 0, 0, 0, time.UTC), // Century non-leap (not divisible by 400)
	}
}
