package util

import "testing"

// These UI-facing state constants are a parallel numbering to model.State and
// are relied upon by the frontend; lock in their integer values.
func TestStateConstants(t *testing.T) {
	cases := []struct {
		got  int
		want int
		name string
	}{
		{Untracked, 0, "Untracked"},
		{WFH, 1, "WFH"},
		{Office, 2, "Office"},
		{Other, 3, "Other"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
}
