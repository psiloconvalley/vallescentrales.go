// internal/repo/session_repo.go
// All SQL for the sessions table.
// Rule 42: repo = SQL only. No business logic. No HTTP.
// Rule 8:  Always parameterized queries. Never string interpolation.
// Rule 9:  Column names verified against live schema 2026-06-20.

package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vallescentrales/internal/models"
)

// SessionRepo handles all database operations for the sessions table.
type SessionRepo struct {
	db *pgxpool.Pool
}

// NewSessionRepo creates a new SessionRepo.
func NewSessionRepo(db *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{db: db}
}

// Create inserts a new session record.
// id must be a crypto/rand generated hex string — see internal/auth/session.go.
func (r *SessionRepo) Create(ctx context.Context, id string, userID uuid.UUID) (*models.Session, error) {
	query := `
		INSERT INTO sessions (id, user_id)
		VALUES ($1, $2)
		RETURNING id, user_id, created_at, expires_at, last_seen`

	session := &models.Session{}
	err := r.db.QueryRow(ctx, query, id, userID).Scan(
		&session.ID,
		&session.UserID,
		&session.CreatedAt,
		&session.ExpiresAt,
		&session.LastSeen,
	)
	if err != nil {
		return nil, fmt.Errorf("session_repo.Create: %w", err)
	}

	return session, nil
}

// GetByID fetches a session by its ID.
// Returns ErrNotFound if the session does not exist.
func (r *SessionRepo) GetByID(ctx context.Context, id string) (*models.Session, error) {
	query := `
		SELECT id, user_id, created_at, expires_at, last_seen
		FROM sessions
		WHERE id = $1`

	session := &models.Session{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&session.ID,
		&session.UserID,
		&session.CreatedAt,
		&session.ExpiresAt,
		&session.LastSeen,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("session_repo.GetByID: %w", err)
	}

	return session, nil
}

// Touch updates last_seen to now for sliding session expiry.
func (r *SessionRepo) Touch(ctx context.Context, id string) error {
	query := `
		UPDATE sessions
		SET last_seen = NOW()
		WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("session_repo.Touch: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete removes a session — called on logout.
func (r *SessionRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM sessions WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("session_repo.Delete: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteAllForUser removes all sessions for a user.
// Used when a user changes their password or is banned.
func (r *SessionRepo) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM sessions WHERE user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("session_repo.DeleteAllForUser: %w", err)
	}

	return nil
}

// DeleteExpired removes all expired sessions.
// Called periodically to keep the sessions table clean.
func (r *SessionRepo) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM sessions WHERE expires_at < NOW()`

	result, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("session_repo.DeleteExpired: %w", err)
	}

	return result.RowsAffected(), nil
}
