// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"github.com/MadsRC/trustedai"
)

// Session represents a user session
type Session struct {
	ID        string
	User      *trustedai.User
	CreatedAt time.Time
	ExpiresAt time.Time
}

// SessionStore defines the interface for session storage
type SessionStore interface {
	// Create creates a new session for the given user
	Create(user *trustedai.User) (*Session, error)

	// Get retrieves a session by ID
	Get(ctx context.Context, id string) (*Session, error)

	// Delete removes a session
	Delete(id string) error

	// Cleanup removes expired sessions
	Cleanup()
}

// MemorySessionStore implements SessionStore using in-memory storage
type MemorySessionStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewMemorySessionStore creates a new in-memory session store
func NewMemorySessionStore() *MemorySessionStore {
	store := &MemorySessionStore{
		sessions: make(map[string]*Session),
	}

	// Start a goroutine to periodically clean up expired sessions
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			store.Cleanup()
		}
	}()

	return store
}

// Create creates a new session for the given user
func (s *MemorySessionStore) Create(user *trustedai.User) (*Session, error) {
	// Generate a random session ID
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	id := base64.URLEncoding.EncodeToString(b)

	// Create a new session
	session := &Session{
		ID:        id,
		User:      user,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // Sessions expire after 24 hours
	}

	// Store the session
	s.mu.Lock()
	s.sessions[id] = session
	s.mu.Unlock()

	return session, nil
}

// Get retrieves a session by ID
func (s *MemorySessionStore) Get(ctx context.Context, id string) (*Session, error) {
	s.mu.RLock()
	session, exists := s.sessions[id]
	s.mu.RUnlock()

	if !exists {
		return nil, errors.New("session not found")
	}

	// Check if the session has expired
	if time.Now().After(session.ExpiresAt) {
		_ = s.Delete(id)
		return nil, errors.New("session expired")
	}

	return session, nil
}

// Delete removes a session
func (s *MemorySessionStore) Delete(id string) error {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
	return nil
}

// Cleanup removes expired sessions
func (s *MemorySessionStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, id)
		}
	}
}
