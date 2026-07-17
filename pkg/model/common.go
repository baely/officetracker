package model

type State int

const (
	StateUntracked = State(iota)
	StateWorkFromHome
	StateWorkFromOffice
	StateOther
	// Scheduled/planned states (lighter versions)
	StateScheduledWorkFromHome
	StateScheduledWorkFromOffice
	StateScheduledOther
)

type DayState struct {
	State State `json:"state"`
}

type MonthState struct {
	Days map[int]DayState `json:"days"`
}

type YearState struct {
	Months map[int]MonthState `json:"months"`
}

type Note struct {
	Note string `json:"note"`
}

// DefaultTrackingYearStartMonth is the month (1-12) a tracking year starts on by
// default. October matches the original hardcoded behaviour.
const DefaultTrackingYearStartMonth = 10

// DefaultTargetPercent is the monthly attendance target applied when a user
// hasn't chosen one, matching the office mandate most users are under. Users
// without a mandate can clear the target in settings (stored as 0).
const DefaultTargetPercent = 50
