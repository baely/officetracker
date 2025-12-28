package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRealClock_Now(t *testing.T) {
	clock := &RealClock{}

	before := time.Now()
	result := clock.Now()
	after := time.Now()

	// Verify the result is between before and after
	assert.True(t, result.After(before) || result.Equal(before))
	assert.True(t, result.Before(after) || result.Equal(after))
}

func TestRealClock_Date(t *testing.T) {
	clock := &RealClock{}

	result := clock.Date(2024, time.October, 15)

	assert.Equal(t, 2024, result.Year())
	assert.Equal(t, time.October, result.Month())
	assert.Equal(t, 15, result.Day())
	assert.Equal(t, 0, result.Hour())
	assert.Equal(t, 0, result.Minute())
	assert.Equal(t, 0, result.Second())
	assert.Equal(t, time.UTC, result.Location())
}
