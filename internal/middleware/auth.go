// internal/middleware/auth.go
// Authentication middleware for protected routes.
// RequireAuth: redirects to /auth/login if no valid session.
// LoadUser: loads the current user into request context.
// Rule 41: if a route is protected, middleware enforces it here.

package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"vallescentrales/internal/auth"
	"vallescentrales/internal/models"
	"vallescentrales/internal/repo"
)

// contextKey is an unexported type for context keys in this package.
// Prevents collisions with other packages using context.
type contextKey string

const (
	// contextKeyUser is the key for the authenticated user in request context.
	contextKeyUser contextKey = "user"
)

// AuthMiddleware holds dependencies for auth middleware functions.
type AuthMiddleware struct {
	sessions *auth.SessionManager
	users    *repo.UserRepo
}

// NewAuthMiddleware creates an AuthMiddleware.
func NewAuthMiddleware(sessions *auth.SessionManager, users *repo.UserRepo) *AuthMiddleware {
	return &AuthMiddleware{
		sessions: sessions,
		users:    users,
	}
}

// LoadUser attempts to load the current user from the session cookie.
// If no valid session exists, the request continues without a user.
// Use this on public routes that optionally show user state.
func (m *AuthMiddleware) LoadUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := m.sessions.Load(r.Context(), r)
		if err != nil {
			// No session — continue as unauthenticated
			next.ServeHTTP(w, r)
			return
		}

		user, err := m.users.GetByID(r.Context(), session.UserID)
		if err != nil {
			if err == repo.ErrNotFound {
				// Session exists but user was deleted — destroy orphan session
				slog.Warn("session references deleted user",
					"user_id", session.UserID,
				)
				_ = m.sessions.Destroy(r.Context(), w, r)
			} else {
				slog.Error("failed to load user from session",
					"user_id", session.UserID,
					"error", err,
				)
			}
			next.ServeHTTP(w, r)
			return
		}

		// User loaded — store in context for handlers to read
		ctx := context.WithValue(r.Context(), contextKeyUser, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuth enforces authentication on protected routes.
// If no valid session exists, redirects to /auth/login.
// Must be applied AFTER LoadUser in the middleware chain.
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			slog.Info("unauthenticated request to protected route",
				"path", r.URL.Path,
				"method", r.Method,
			)
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// UserFromContext retrieves the authenticated user from the request context.
// Returns nil if no user is present — always check before using.
func UserFromContext(ctx context.Context) *models.User {
	user, _ := ctx.Value(contextKeyUser).(*models.User)
	return user
}
