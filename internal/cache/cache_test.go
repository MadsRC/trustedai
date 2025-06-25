// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !integration && !acceptance

package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache_BasicOperations(t *testing.T) {
	cache := New[string, string](time.Minute)
	defer cache.Close()

	// Test Set and Get
	cache.Set("key1", "value1")
	value, found := cache.Get("key1")
	require.True(t, found)
	assert.Equal(t, "value1", value)

	// Test Get non-existent key
	_, found = cache.Get("nonexistent")
	assert.False(t, found)

	// Test Size
	cache.Set("key2", "value2")
	assert.Equal(t, 2, cache.Size())

	// Test Delete
	cache.Delete("key1")
	_, found = cache.Get("key1")
	assert.False(t, found)
	assert.Equal(t, 1, cache.Size())

	// Test Clear
	cache.Clear()
	assert.Equal(t, 0, cache.Size())
}

func TestCache_TTL(t *testing.T) {
	cache := New[string, string](100 * time.Millisecond)
	defer cache.Close()

	// Set a value
	cache.Set("key1", "value1")

	// Should be available immediately
	value, found := cache.Get("key1")
	require.True(t, found)
	assert.Equal(t, "value1", value)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, found = cache.Get("key1")
	assert.False(t, found)
}

func TestCache_EntryExpiration(t *testing.T) {
	entry := &Entry[string]{
		Value:     "test",
		ExpiresAt: time.Now().Add(-time.Minute), // Already expired
	}

	assert.True(t, entry.IsExpired())

	entry.ExpiresAt = time.Now().Add(time.Minute)
	assert.False(t, entry.IsExpired())
}

func TestCache_Cleanup(t *testing.T) {
	cache := New[string, string](50 * time.Millisecond)
	defer cache.Close()

	// Add entries
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	assert.Equal(t, 2, cache.Size())

	// Wait for expiration and cleanup
	time.Sleep(100 * time.Millisecond)

	// Size should eventually be 0 as cleanup runs
	// Note: This test may be flaky due to timing, but it demonstrates the concept
	assert.Eventually(t, func() bool {
		return cache.Size() == 0
	}, time.Second, 10*time.Millisecond)
}
