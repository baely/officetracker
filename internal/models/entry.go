package models

import "time"

type Entry struct {
	User        string
	CreateDate  time.Time
	Month, Year int
	Days        map[string]int
	Notes       string
}

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
