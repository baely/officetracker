package auth_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/baely/officetracker/internal/auth"
)

func TestGenerateSecret(t *testing.T) {
	t.Run("generate", func(t *testing.T) {
		got := auth.GenerateSecret()

		require.Len(t, got, 78)
		require.Equal(t, "officetracker:", got[:14])
	})
}
