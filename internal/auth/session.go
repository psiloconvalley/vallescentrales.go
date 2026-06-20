// internal/auth/session.go
// Session lifecycle: create, load, destroy.
// ADR-003: server-side sessions backed by PostgreSQL.
// Security: httpOnly, SameSite=Strict, Secure in production.

package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"vallescentrales/internal/models"
	"vallescentrales/internal/repo"
)

const (
	// SessionCookieName is the httpOnly session cookie name.
	SessionCookieName = "vc_session"

	// sessionIDLength is 32 random bytes = 64 hex chars = 256 bits entropy.
	sessionIDLength = 32
)

// ErrNoSession is returned when no valid session exists for a request.
var ErrNoSession = errors.New("no session")

// SessionManager handles session creation, loading, and destruction.
type SessionManager struct {
	sessions *repo.SessionRepo
	secure   bool // true in production — enforces HTTPS-only cookie
}

// NewSessionManager creates a SessionManager.
// secure must be true in production, false in local development.
func NewSessionManager(sessions *repo.SessionRepo, secure bool) *SessionManager {
	return &SessionManager{
		sessions: sessions,
		secure:   secure,
	}
}

// Create generates a new session for a user and sets the session cookie.
// Called immediately after successful login or registration.
func (sm *SessionManager) Create(ctx context.Context, w http.ResponseWriter, userID uuid.UUID) (*models.Session, error) {
	id, err := generateSessionID()
	if err != nil {
		slog.Error("failed to generate session ID", "error", err)
		return nil, fmt.Errorf("auth.SessionManager.Create: %w", err)
	}

	session, err := sm.sessions.Create(ctx, id, userID)
	if err != nil {
		slog.Error("failed to persist session", "user_id", userID, "error", err)
		return nil, fmt.Errorf("auth.SessionManager.Create: %w", err)
	}

	sm.setCookie(w, session.ID)

	slog.Info("session created", "user_id", userID, "session_id_prefix", session.ID[:8])

	return session, nil
}

// Load reads the session cookie, fetches the session, and validates expiry.
// Returns ErrNoSession if missing, expired, or not found.
// Silently cleans up expired sessions.
func (sm *SessionManager) Load(ctx context.Context, r *http.Request) (*models.Session, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		// No cookie present — unauthenticated request, not an error
		return nil, ErrNoSession
	}

	// Validate session ID length before hitting the DB
	if len(cookie.Value) != sessionIDLength*2 {
		slog.Warn("session cookie has unexpected length",
			"length", len(cookie.Value),
			"expected", sessionIDLength*2,
		)
		return nil, ErrNoSession
	}

	session, err := sm.sessions.GetByID(ctx, cookie.Value)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNoSession
		}
		slog.Error("failed to load session from DB", "error", err)
		return nil, fmt.Errorf("auth.SessionManager.Load: %w", err)
	}

	if session.IsExpired() {
		slog.Info("expired session cleaned up", "session_id_prefix", session.ID[:8])
		if err := sm.sessions.Delete(ctx, session.ID); err != nil {
			slog.Warn("failed to delete expired session", "error", err)
		}
		return nil, ErrNoSession
	}

	// Touch updates last_seen for sliding expiry window
	if err := sm.sessions.Touch(ctx, session.ID); err != nil {
		// Non-fatal — session is still valid, just log it
		slog.Warn("failed to touch session last_seen", "error", err)
	}

	return session, nil
}

// Destroy deletes the session from DB and clears the cookie.
// Called on logout. Always clears the cookie even if DB delete fails.
func (sm *SessionManager) Destroy(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		// No cookie — nothing to destroy
		return nil
	}

	if err := sm.sessions.Delete(ctx, cookie.Value); err != nil {
		if !errors.Is(err, repo.ErrNotFound) {
			slog.Error("failed to delete session from DB on logout", "error", err)
		}
	}

	// Always clear the cookie — even if DB delete failed
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   sm.secure,
		SameSite: http.SameSiteStrictMode,
	})

	slog.Info("session destroyed")

	return nil
}

// setCookie writes the session cookie to the HTTP response.
// httpOnly:  JS cannot read it — XSS cannot steal sessions
// Secure:    HTTPS only in production
// SameSite:  Strict — CSRF protection at cookie level
func (sm *SessionManager) setCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int((30 * 24 * time.Hour).Seconds()),
		HttpOnly: true,
		Secure:   sm.secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// generateSessionID creates a cryptographically secure random hex string.
// 32 bytes of entropy = 256 bits = unguessable.
func generateSessionID() (string, error) {
	b := make([]byte, sessionIDLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generateSessionID: failed to read random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}
