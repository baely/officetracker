package util

import (
	"os"
	"testing"
)

func TestLoadEnv(t *testing.T) {
	// Save and restore original APP_ENV
	originalEnv := os.Getenv("APP_ENV")
	defer func() {
		if originalEnv != "" {
			os.Setenv("APP_ENV", originalEnv)
		} else {
			os.Unsetenv("APP_ENV")
		}
	}()

	// Set a test environment that doesn't exist
	os.Setenv("APP_ENV", "nonexistent")

	// This should not panic even if the file doesn't exist
	LoadEnv()

	// Test passes if no panic occurs
}

func TestLoadEnv_NoAppEnv(t *testing.T) {
	// Save and restore original APP_ENV
	originalEnv := os.Getenv("APP_ENV")
	defer func() {
		if originalEnv != "" {
			os.Setenv("APP_ENV", originalEnv)
		} else {
			os.Unsetenv("APP_ENV")
		}
	}()

	// Unset APP_ENV
	os.Unsetenv("APP_ENV")

	// This should not panic
	LoadEnv()

	// Test passes if no panic occurs
}
