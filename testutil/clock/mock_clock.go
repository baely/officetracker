package clock

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

// MockClock is a test implementation with controllable time
type MockClock struct {
	currentTime time.Time
}

// NewMockClock creates a MockClock set to the current time
func NewMockClock() *MockClock {
	return &MockClock{
		currentTime: time.Now(),
	}
}

// NewMockClockAt creates a MockClock set to a specific date at noon UTC
func NewMockClockAt(year int, month time.Month, day int) *MockClock {
	return &MockClock{
		currentTime: time.Date(year, month, day, 12, 0, 0, 0, time.UTC),
	}
}

// Now returns the current mocked time
func (m *MockClock) Now() time.Time {
	return m.currentTime
}

// Date creates a time.Time for the given date at midnight UTC
func (m *MockClock) Date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// SetTime sets the current time to the specified value
func (m *MockClock) SetTime(t time.Time) {
	m.currentTime = t
}

// Advance moves the clock forward by the specified duration
func (m *MockClock) Advance(d time.Duration) {
	m.currentTime = m.currentTime.Add(d)
}

// Verify interface compliance
var _ Clock = (*RealClock)(nil)
var _ Clock = (*MockClock)(nil)
