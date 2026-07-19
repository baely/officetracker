package util

import "testing"

func TestClampTargetPercent(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{-5, 0},
		{0, 0},
		{1, 1},
		{50, 50},
		{100, 100},
		{101, 100},
		{1000, 100},
	}
	for _, c := range cases {
		if got := ClampTargetPercent(c.in); got != c.want {
			t.Errorf("ClampTargetPercent(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}
