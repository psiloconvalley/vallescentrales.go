// internal/handlers/auth_handler.go
// Handles registration, login, logout, and Google OAuth.
// Rule 42: handlers = HTTP only. No SQL. No business logic.

package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"vallescentrales/internal/auth"
	"vallescentrales/internal/middleware"
	"vallescentrales/internal/repo"
)

// AuthHandler handles all authentication HTTP endpoints.
type AuthHandler struct {
	users      *repo.UserRepo
	sessions   *auth.SessionManager
	googleAuth *auth.GoogleOAuth
}

// NewAuthHandler creates an AuthHandler.
// googleAuth may be nil if Google OAuth is not configured.
func NewAuthHandler(users *repo.UserRepo, sessions *auth.SessionManager, googleAuth *auth.GoogleOAuth) *AuthHandler {
	return &AuthHandler{
		users:      users,
		sessions:   sessions,
		googleAuth: googleAuth,
	}
}

// GoogleEnabled returns true if Google OAuth is available.
func (h *AuthHandler) GoogleEnabled() bool {
	return h.googleAuth != nil && h.googleAuth.Enabled()
}

// HandleRegisterPage serves the registration form.
func (h *AuthHandler) HandleRegisterPage(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]any{
		"page":          "register",
		"google_enabled": h.GoogleEnabled(),
	})
}

// HandleRegister processes a new user registration via email + password.
func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		respondError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	email    := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")
	fullName := strings.TrimSpace(r.FormValue("full_name"))

	if email == "" || password == "" || fullName == "" {
		respondError(w, http.StatusBadRequest, "email, password, and full name are required")
		return
	}

	if !strings.Contains(email, "@") {
		respondError(w, http.StatusBadRequest, "invalid email address")
		return
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		if errors.Is(err, auth.ErrPasswordTooShort) {
			respondError(w, http.StatusBadRequest, "password must be at least 12 characters")
			return
		}
		if errors.Is(err, auth.ErrPasswordTooLong) {
			respondError(w, http.StatusBadRequest, "password must be 72 characters or fewer")
			return
		}
		slog.Error("failed to hash password during registration", "error", err)
		respondError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	user, err := h.users.Create(r.Context(), email, passwordHash, fullName)
	if err != nil {
		if errors.Is(err, repo.ErrEmailTaken) {
			respondError(w, http.StatusConflict, "an account with that email already exists")
			return
		}
		slog.Error("failed to create user", "email", email, "error", err)
		respondError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	_, err = h.sessions.Create(r.Context(), w, user.ID)
	if err != nil {
		slog.Error("failed to create session after registration",
			"user_id", user.ID, "error", err,
		)
		respondError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	slog.Info("user registered", "user_id", user.ID, "email", user.Email, "provider", "email")

	respond(w, http.StatusCreated, user.ToSafe())
}

// HandleLoginPage serves the login form.
func (h *AuthHandler) HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]any{
		"page":          "login",
		"google_enabled": h.GoogleEnabled(),
	})
}

// HandleLogin processes a login attempt via email + password.
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		respondError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	email    := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	if email == "" || password == "" {
		respondError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := h.users.GetByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			respondError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		slog.Error("failed to fetch user during login", "email", email, "error", err)
		respondError(w, http.StatusInternalServerError, "login failed")
		return
	}

	if !user.HasPassword() {
		slog.Info("email login attempt on passwordless account", "email", email)
		respondError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if err := auth.VerifyPassword(password, *user.PasswordHash); err != nil {
		slog.Info("failed login attempt", "email", email)
		respondError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	_, err = h.sessions.Create(r.Context(), w, user.ID)
	if err != nil {
		slog.Error("failed to create session after login",
			"user_id", user.ID, "error", err,
		)
		respondError(w, http.StatusInternalServerError, "login failed")
		return
	}

	slog.Info("user logged in", "user_id", user.ID, "email", user.Email, "provider", "email")

	respond(w, http.StatusOK, user.ToSafe())
}

// HandleLogout destroys the session and clears the cookie.
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	if err := h.sessions.Destroy(r.Context(), w, r); err != nil {
		slog.Error("failed to destroy session on logout", "error", err)
	}

	if user != nil {
		slog.Info("user logged out", "user_id", user.ID)
	}

	respond(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// HandleGoogleLogin redirects to Google's consent screen.
func (h *AuthHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if !h.GoogleEnabled() {
		respondError(w, http.StatusNotImplemented, "google login not configured")
		return
	}

	h.googleAuth.RedirectToGoogle(w, r)
}

// HandleGoogleCallback processes the redirect back from Google.
// Creates or finds the user, links accounts if needed, starts session.
func (h *AuthHandler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if !h.GoogleEnabled() {
		respondError(w, http.StatusNotImplemented, "google login not configured")
		return
	}

	googleUser, err := h.googleAuth.ProcessCallback(w, r)
	if err != nil {
		slog.Error("google oauth callback failed", "error", err)
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	ctx := r.Context()

	// Try to find existing user by Google ID
	user, err := h.users.GetByGoogleID(ctx, googleUser.ID)
	if err == nil {
		// Existing Google user — start session
		_, err = h.sessions.Create(ctx, w, user.ID)
		if err != nil {
			slog.Error("failed to create session for google user",
				"user_id", user.ID, "error", err,
			)
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		slog.Info("user logged in via google", "user_id", user.ID, "email", user.Email)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	// Check if email already exists (registered via email)
	existingUser, err := h.users.GetByEmail(ctx, googleUser.Email)
	if err == nil {
		// Link Google account to existing email user
		user, err = h.users.LinkGoogleAccount(ctx, existingUser.ID, googleUser.ID)
		if err != nil {
			slog.Error("failed to link google account",
				"user_id", existingUser.ID, "error", err,
			)
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		_, err = h.sessions.Create(ctx, w, user.ID)
		if err != nil {
			slog.Error("failed to create session after google link",
				"user_id", user.ID, "error", err,
			)
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		slog.Info("google account linked to existing user",
			"user_id", user.ID, "email", user.Email,
		)
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	// New user — create with Google provider
	user, err = h.users.CreateGoogle(ctx, googleUser.Email, googleUser.Name, googleUser.ID)
	if err != nil {
		slog.Error("failed to create google user",
			"email", googleUser.Email, "error", err,
		)
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	_, err = h.sessions.Create(ctx, w, user.ID)
	if err != nil {
		slog.Error("failed to create session for new google user",
			"user_id", user.ID, "error", err,
		)
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	slog.Info("user registered via google",
		"user_id", user.ID, "email", user.Email,
	)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
