package stats

import "testing"

// formatInt renders dashboard numbers with thousands separators. Cover the
// grouping boundaries and the negative-number path.
func TestFormatInt(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{5, "5"},
		{42, "42"},
		{100, "100"},
		{999, "999"},
		{1000, "1,000"},
		{1234, "1,234"},
		{12345, "12,345"},
		{123456, "123,456"},
		{1234567, "1,234,567"},
		{1000000, "1,000,000"},
		{-1, "-1"},
		{-1234, "-1,234"},
		{-1000000, "-1,000,000"},
	}
	for _, c := range cases {
		if got := formatInt(c.in); got != c.want {
			t.Errorf("formatInt(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
