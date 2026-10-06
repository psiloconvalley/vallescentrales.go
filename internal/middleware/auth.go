// internal/middleware/auth.go
// Authentication middleware for protected routes.
// RequireAuth: redirects to /login if no valid session.
// LoadUser: loads the current user into request context.

package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"

	"vallescentrales/internal/auth"
	"vallescentrales/internal/models"
	"vallescentrales/internal/repo"
)

type contextKey string

const (
	contextKeyUser contextKey = "user"
)

type AuthMiddleware struct {
	sessions *auth.SessionManager
	users    *repo.UserRepo
}

func NewAuthMiddleware(sessions *auth.SessionManager, users *repo.UserRepo) *AuthMiddleware {
	return &AuthMiddleware{
		sessions: sessions,
		users:    users,
	}
}

func (m *AuthMiddleware) LoadUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := m.sessions.Load(r.Context(), r)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		user, err := m.users.GetByID(r.Context(), session.UserID)
		if err != nil {
			if err == repo.ErrNotFound {
				slog.Warn("session references deleted user", "user_id", session.UserID)
				_ = m.sessions.Destroy(r.Context(), w, r)
			} else {
				slog.Error("failed to load user from session", "user_id", session.UserID, "error", err)
			}
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), contextKeyUser, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil {
			slog.Info("unauthenticated request to protected route",
				"path", r.URL.Path,
				"method", r.Method,
			)
			
			// Build target redirection path with safe URL escaping
			redirectURL := "/login"
			if r.URL.Path != "" && r.URL.Path != "/" {
				redirectURL += "?redirect=" + url.QueryEscape(r.URL.Path)
			}
			
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func UserFromContext(ctx context.Context) *models.User {
	user, _ := ctx.Value(contextKeyUser).(*models.User)
	return user
}
