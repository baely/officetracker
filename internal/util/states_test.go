package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStateConstants(t *testing.T) {
	tests := []struct {
		name     string
		state    int
		expected int
	}{
		{"Untracked", Untracked, 0},
		{"WFH", WFH, 1},
		{"Office", Office, 2},
		{"Other", Other, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state)
		})
	}
}

func TestStateConstantsAreSequential(t *testing.T) {
	// Verify constants are sequential starting from 0
	assert.Equal(t, 0, Untracked)
	assert.Equal(t, 1, WFH)
	assert.Equal(t, 2, Office)
	assert.Equal(t, 3, Other)
}
