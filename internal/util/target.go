package util

// ClampTargetPercent normalises an attendance target percentage to the valid
// 0-100 range, where 0 means no target is set.
func ClampTargetPercent(percent int) int {
	if percent < 0 {
		return 0
	}
	if percent > 100 {
		return 100
	}
	return percent
}
