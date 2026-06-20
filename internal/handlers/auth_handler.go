// internal/handlers/auth_handler.go
// Handles registration, login, and logout.
// Rule 42: handlers = HTTP only. No SQL. No business logic.
// Business logic lives in auth package. SQL lives in repo package.

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
	users    *repo.UserRepo
	sessions *auth.SessionManager
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(users *repo.UserRepo, sessions *auth.SessionManager) *AuthHandler {
	return &AuthHandler{
		users:    users,
		sessions: sessions,
	}
}

// HandleRegisterPage serves the registration form.
// Returns JSON stub — replaced with template in Session 010.
func (h *AuthHandler) HandleRegisterPage(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{"page": "register"})
}

// HandleRegister processes a new user registration.
// Validates input, hashes password with Argon2id, creates user, starts session.
func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		respondError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	email    := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")
	fullName := strings.TrimSpace(r.FormValue("full_name"))

	// Input validation
	if email == "" || password == "" || fullName == "" {
		respondError(w, http.StatusBadRequest, "email, password, and full name are required")
		return
	}

	if !strings.Contains(email, "@") {
		respondError(w, http.StatusBadRequest, "invalid email address")
		return
	}

	// Hash password — Argon2id, validates length internally
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

	// Create user
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

	// Start session immediately after registration
	_, err = h.sessions.Create(r.Context(), w, user.ID)
	if err != nil {
		slog.Error("failed to create session after registration",
			"user_id", user.ID,
			"error", err,
		)
		respondError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	slog.Info("user registered", "user_id", user.ID, "email", user.Email)

	// Session 010: replace with http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	respond(w, http.StatusCreated, user.ToSafe())
}

// HandleLoginPage serves the login form.
// Returns JSON stub — replaced with template in Session 010.
func (h *AuthHandler) HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{"page": "login"})
}

// HandleLogin processes a login attempt.
// Fetches user, verifies Argon2id hash, creates session on success.
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

	// Fetch user — use generic error message to not leak user existence
	user, err := h.users.GetByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			// Do not reveal whether the email exists
			respondError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		slog.Error("failed to fetch user during login", "email", email, "error", err)
		respondError(w, http.StatusInternalServerError, "login failed")
		return
	}

	// Verify password — constant-time comparison inside
	if err := auth.VerifyPassword(password, user.PasswordHash); err != nil {
		slog.Info("failed login attempt", "email", email)
		respondError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	// Start session
	_, err = h.sessions.Create(r.Context(), w, user.ID)
	if err != nil {
		slog.Error("failed to create session after login",
			"user_id", user.ID,
			"error", err,
		)
		respondError(w, http.StatusInternalServerError, "login failed")
		return
	}

	slog.Info("user logged in", "user_id", user.ID, "email", user.Email)

	// Session 010: replace with http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
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

	// Session 010: replace with http.Redirect(w, r, "/", http.StatusSeeOther)
	respond(w, http.StatusOK, map[string]string{"message": "logged out"})
}
