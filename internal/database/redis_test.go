package database

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/baely/officetracker/internal/config"
)

func setupTestRedis(t *testing.T) (*Redis, *miniredis.Miniredis) {
	// Create miniredis server
	server := miniredis.RunT(t)

	cfg := config.Redis{
		Host: server.Addr(),
	}

	redis, err := NewRedis(cfg)
	require.NoError(t, err)

	return redis, server
}

func TestNewRedis(t *testing.T) {
	server := miniredis.RunT(t)
	defer server.Close()

	cfg := config.Redis{
		Host: server.Addr(),
	}

	redis, err := NewRedis(cfg)
	require.NoError(t, err)
	assert.NotNil(t, redis)
	assert.NotNil(t, redis.rdb)
}

func TestRedis_SetAndGetStateInt(t *testing.T) {
	redis, server := setupTestRedis(t)
	defer server.Close()

	ctx := context.Background()
	key := "test:key"
	value := 42

	// Set state
	err := redis.SetState(ctx, key, value, 10*time.Minute)
	require.NoError(t, err)

	// Get state
	retrievedValue, err := redis.GetStateInt(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrievedValue)
}

func TestRedis_SetStateWithExpiration(t *testing.T) {
	redis, server := setupTestRedis(t)
	defer server.Close()

	ctx := context.Background()
	key := "test:expiring"
	value := 123

	// Set state with expiration
	err := redis.SetState(ctx, key, value, 1*time.Second)
	require.NoError(t, err)

	// Verify it exists
	retrievedValue, err := redis.GetStateInt(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrievedValue)

	// Fast-forward time in miniredis
	server.FastForward(2 * time.Second)

	// Verify it's gone
	_, err = redis.GetStateInt(ctx, key)
	assert.Error(t, err)
}

func TestRedis_GetStateInt_NotFound(t *testing.T) {
	redis, server := setupTestRedis(t)
	defer server.Close()

	ctx := context.Background()

	// Try to get non-existent key
	_, err := redis.GetStateInt(ctx, "nonexistent:key")
	assert.Error(t, err)
}

func TestRedis_DeleteState(t *testing.T) {
	redis, server := setupTestRedis(t)
	defer server.Close()

	ctx := context.Background()
	key := "test:deleteme"
	value := 99

	// Set state
	err := redis.SetState(ctx, key, value, 10*time.Minute)
	require.NoError(t, err)

	// Verify it exists
	retrievedValue, err := redis.GetStateInt(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrievedValue)

	// Delete state
	err = redis.DeleteState(ctx, key)
	require.NoError(t, err)

	// Verify it's gone
	_, err = redis.GetStateInt(ctx, key)
	assert.Error(t, err)
}

func TestRedis_DeleteState_NonExistent(t *testing.T) {
	redis, server := setupTestRedis(t)
	defer server.Close()

	ctx := context.Background()

	// Delete non-existent key (should not error)
	err := redis.DeleteState(ctx, "nonexistent:key")
	require.NoError(t, err)
}
