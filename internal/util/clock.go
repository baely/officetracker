package util

import "time"

// Clock interface for testable time operations
type Clock interface {
	Now() time.Time
	Date(year int, month time.Month, day int) time.Time
}

// RealClock is the production implementation that uses actual time
type RealClock struct{}

// Now returns the current time
func (c *RealClock) Now() time.Time {
	return time.Now()
}

// Date creates a time.Time for the given date at midnight UTC
func (c *RealClock) Date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
