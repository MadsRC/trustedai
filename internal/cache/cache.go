// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package cache

import (
	"sync"
	"time"
)

// Entry represents a cached entry with expiration
type Entry[T any] struct {
	Value     T
	ExpiresAt time.Time
}

// IsExpired checks if the cache entry has expired
func (e *Entry[T]) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// Cache is a generic in-memory cache with TTL support
type Cache[K comparable, V any] struct {
	mu       sync.RWMutex
	entries  map[K]*Entry[V]
	ttl      time.Duration
	stopChan chan struct{}
	once     sync.Once
}

// New creates a new cache with the specified TTL
func New[K comparable, V any](ttl time.Duration) *Cache[K, V] {
	c := &Cache[K, V]{
		entries:  make(map[K]*Entry[V]),
		ttl:      ttl,
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine
	go c.cleanupLoop()

	return c
}

// Get retrieves a value from the cache
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists || entry.IsExpired() {
		var zero V
		return zero, false
	}

	return entry.Value, true
}

// Set stores a value in the cache with TTL
func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &Entry[V]{
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes a value from the cache
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}

// Clear removes all entries from the cache
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[K]*Entry[V])
}

// Size returns the number of entries in the cache
func (c *Cache[K, V]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

// Close stops the cleanup goroutine
func (c *Cache[K, V]) Close() {
	c.once.Do(func() {
		close(c.stopChan)
	})
}

// cleanupLoop periodically removes expired entries
func (c *Cache[K, V]) cleanupLoop() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopChan:
			return
		}
	}
}

// cleanup removes expired entries
func (c *Cache[K, V]) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
}
