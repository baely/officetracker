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
