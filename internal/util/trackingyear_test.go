package util

import (
	"testing"
	"time"
)

func TestTrackingYear(t *testing.T) {
	cases := []struct {
		name       string
		month, cy  int
		startMonth int
		want       int
	}{
		// October start (default, must match original behaviour).
		{"oct: sep stays", 9, 2024, 10, 2024},
		{"oct: oct rolls", 10, 2023, 10, 2024},
		{"oct: dec rolls", 12, 2023, 10, 2024},
		{"oct: jan stays", 1, 2024, 10, 2024},
		// January start == calendar year.
		{"jan: jan", 1, 2024, 1, 2024},
		{"jan: dec", 12, 2024, 1, 2024},
		// July start.
		{"jul: jun stays", 6, 2024, 7, 2024},
		{"jul: jul rolls", 7, 2023, 7, 2024},
		// Out of range falls back to October default.
		{"bad: treated as oct", 10, 2023, 0, 2024},
	}
	for _, c := range cases {
		if got := TrackingYear(c.month, c.cy, c.startMonth); got != c.want {
			t.Errorf("%s: TrackingYear(%d,%d,%d)=%d want %d", c.name, c.month, c.cy, c.startMonth, got, c.want)
		}
	}
}

func TestTrackingYearCalendarYears(t *testing.T) {
	cases := []struct {
		ty, startMonth        int
		wantFirst, wantSecond int
	}{
		{2024, 10, 2023, 2024},
		{2024, 1, 2024, 2024},
		{2024, 7, 2023, 2024},
	}
	for _, c := range cases {
		first, second := TrackingYearCalendarYears(c.ty, c.startMonth)
		if first != c.wantFirst || second != c.wantSecond {
			t.Errorf("TrackingYearCalendarYears(%d,%d)=(%d,%d) want (%d,%d)", c.ty, c.startMonth, first, second, c.wantFirst, c.wantSecond)
		}
	}
}

func TestTrackingYearRange(t *testing.T) {
	cases := []struct {
		ty, startMonth int
		wantStart      time.Time
		wantEnd        time.Time
	}{
		{2024, 10, time.Date(2023, time.October, 1, 0, 0, 0, 0, time.Local), time.Date(2024, time.October, 1, 0, 0, 0, 0, time.Local)},
		{2024, 1, time.Date(2024, time.January, 1, 0, 0, 0, 0, time.Local), time.Date(2025, time.January, 1, 0, 0, 0, 0, time.Local)},
		{2024, 7, time.Date(2023, time.July, 1, 0, 0, 0, 0, time.Local), time.Date(2024, time.July, 1, 0, 0, 0, 0, time.Local)},
	}
	for _, c := range cases {
		start, end := TrackingYearRange(c.ty, c.startMonth)
		if !start.Equal(c.wantStart) || !end.Equal(c.wantEnd) {
			t.Errorf("TrackingYearRange(%d,%d)=(%v,%v) want (%v,%v)", c.ty, c.startMonth, start, end, c.wantStart, c.wantEnd)
		}
	}
}
