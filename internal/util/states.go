package util

const (
	Untracked = iota
	WFH
	Office
	Other
)

func StateToString(state int) string {
	switch state {
	case WFH:
		return "Work from Home"
	case Office:
		return "In office"
	case Other:
		return "Other"
	default:
		return "Untracked"
	}
}
