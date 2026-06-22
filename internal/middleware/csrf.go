// internal/middleware/csrf.go
// CSRF protection using nosurf.
// Generates and validates CSRF tokens on all POST requests.
// Token is injected into templates for form rendering.

package middleware

import (
	"net/http"

	"github.com/justinas/nosurf"
)

// CSRFProtect wraps the handler with CSRF token generation and validation.
// All POST requests without a valid token are rejected with 403.
// The token is available in templates via CSRFToken.
func CSRFProtect(secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		csrfHandler := nosurf.New(next)

		csrfHandler.SetBaseCookie(http.Cookie{
			Path:     "/",
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
		})

		// Exempt OAuth callback — Google redirects with GET, no token possible
		csrfHandler.ExemptPath("/auth/google/callback")

		// Exempt health check
		csrfHandler.ExemptPath("/health")

		csrfHandler.SetFailureHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"CSRF token invalid or missing"}`))
		}))

		return csrfHandler
	}
}

// CSRFToken returns the CSRF token for the current request.
// Use this in templates: <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
func CSRFToken(r *http.Request) string {
	return nosurf.Token(r)
}
