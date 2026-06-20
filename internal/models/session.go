// internal/models/session.go
// Pure Go types for the sessions table.
// Derived from migrations/000002_create_sessions.up.sql
// No DB code. No HTTP code. No business logic.

package models

import (
	"time"

	"github.com/google/uuid"
)

// Session maps exactly to the sessions table.
// ID is a crypto/rand generated hex string — not a UUID.
// See internal/auth/session.go for generation logic.
type Session struct {
	ID        string    `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
	ExpiresAt time.Time `db:"expires_at"`
	LastSeen  time.Time `db:"last_seen"`
}

// IsExpired returns true if the session is past its expiry time.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
