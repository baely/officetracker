package auth

import (
	"strings"
	"testing"
)

// GenerateSecret returns an "officetracker:"-prefixed token of a fixed length,
// drawn from a known alphabet, and different on each call.
func TestGenerateSecret(t *testing.T) {
	const prefix = "officetracker:"
	const bodyLen = 64

	s := GenerateSecret()
	if !strings.HasPrefix(s, prefix) {
		t.Fatalf("secret %q missing prefix %q", s, prefix)
	}
	if len(s) != len(prefix)+bodyLen {
		t.Fatalf("secret length = %d, want %d", len(s), len(prefix)+bodyLen)
	}

	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890+/"
	body := strings.TrimPrefix(s, prefix)
	for _, c := range body {
		if !strings.ContainsRune(alphabet, c) {
			t.Errorf("secret body contains unexpected rune %q", c)
		}
	}

	// Successive secrets should differ.
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		v := GenerateSecret()
		if seen[v] {
			t.Fatalf("duplicate secret generated: %q", v)
		}
		seen[v] = true
	}
}
