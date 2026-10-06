// internal/auth/session.go

package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"vallescentrales/internal/models"
	"vallescentrales/internal/repo"
)

const (
	SessionCookieName = "vc_session"
	sessionIDLength   = 32
)

var ErrNoSession = errors.New("no session")

type SessionManager struct {
	sessions *repo.SessionRepo
	secure   bool
	domain   string
}

func NewSessionManager(sessions *repo.SessionRepo, secure bool, baseDomain string) *SessionManager {
	cookieDomain := ""
	// If we are not on localhost/127.0.0.1, always scope the cookie to the wildcard base domain
	if baseDomain != "" && baseDomain != "localhost" && baseDomain != "127.0.0.1" {
		if idx := strings.Index(baseDomain, ":"); idx != -1 {
			baseDomain = baseDomain[:idx]
		}
		if !strings.HasPrefix(baseDomain, ".") {
			cookieDomain = "." + baseDomain
		} else {
			cookieDomain = baseDomain
		}
	}

	return &SessionManager{
		sessions: sessions,
		secure:   secure,
		domain:   cookieDomain,
	}
}

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
	return session, nil
}

func (sm *SessionManager) Load(ctx context.Context, r *http.Request) (*models.Session, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil, ErrNoSession
	}

	if len(cookie.Value) != sessionIDLength*2 {
		return nil, ErrNoSession
	}

	session, err := sm.sessions.GetByID(ctx, cookie.Value)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNoSession
		}
		return nil, fmt.Errorf("auth.SessionManager.Load: %w", err)
	}

	if session.IsExpired() {
		_ = sm.sessions.Delete(ctx, session.ID)
		return nil, ErrNoSession
	}

	_ = sm.sessions.Touch(ctx, session.ID)
	return session, nil
}

func (sm *SessionManager) Destroy(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil
	}

	_ = sm.sessions.Delete(ctx, cookie.Value)

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Domain:   sm.domain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   sm.secure,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

func (sm *SessionManager) setCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		Domain:   sm.domain,
		MaxAge:   int((30 * 24 * time.Hour).Seconds()),
		HttpOnly: true,
		Secure:   sm.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func generateSessionID() (string, error) {
	b := make([]byte, sessionIDLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generateSessionID: failed to read random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}
