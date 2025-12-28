package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSecret(t *testing.T) {
	secret := GenerateSecret()

	// Verify it starts with the prefix
	prefix := "officetracker:"
	assert.True(t, len(secret) > len(prefix), "Secret should be longer than prefix")
	assert.Equal(t, prefix, secret[:len(prefix)], "Secret should start with prefix")

	// Verify total length (prefix + 64 random chars)
	expectedLength := len(prefix) + 64
	assert.Equal(t, expectedLength, len(secret), "Secret should be prefix + 64 characters")

	// Verify the random part only contains valid characters
	randomPart := secret[len(prefix):]
	validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890+/"
	for _, char := range randomPart {
		assert.Contains(t, validChars, string(char), "Random part should only contain valid characters")
	}
}

func TestGenerateSecret_Uniqueness(t *testing.T) {
	// Generate multiple secrets and verify they're unique
	secrets := make(map[string]bool)
	for i := 0; i < 100; i++ {
		secret := GenerateSecret()
		assert.False(t, secrets[secret], "Each generated secret should be unique")
		secrets[secret] = true
	}
	assert.Equal(t, 100, len(secrets), "Should have generated 100 unique secrets")
}
