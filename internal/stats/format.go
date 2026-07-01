package stats

import "strconv"

// formatInt renders an integer with thousands separators (e.g. 1234567 ->
// "1,234,567") for friendly display on the dashboard.
func formatInt(n int) string {
	s := strconv.Itoa(n)
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	// Insert commas every 3 digits from the right.
	out := make([]byte, 0, len(s)+len(s)/3)
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}
