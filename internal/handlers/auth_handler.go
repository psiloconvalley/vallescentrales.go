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
	render     Renderer
}

// NewAuthHandler creates an AuthHandler.
// googleAuth may be nil if Google OAuth is not configured.
// render may be nil — falls back to JSON responses.
func NewAuthHandler(users *repo.UserRepo, sessions *auth.SessionManager, googleAuth *auth.GoogleOAuth, render Renderer) *AuthHandler {
	return &AuthHandler{
		users:      users,
		sessions:   sessions,
		googleAuth: googleAuth,
		render:     render,
	}
}

// GoogleEnabled returns true if Google OAuth is available.
func (h *AuthHandler) GoogleEnabled() bool {
	return h.googleAuth != nil && h.googleAuth.Enabled()
}

// renderPage renders an HTML template if renderer is available, otherwise JSON.
func (h *AuthHandler) renderPage(w http.ResponseWriter, r *http.Request, tmpl string, data map[string]any) {
	if data == nil {
		data = make(map[string]any)
	}

	// Always inject user and google state into template data
	data["User"] = middleware.UserFromContext(r.Context())
	data["GoogleEnabled"] = h.GoogleEnabled()

	if data["Meta"] == nil {
		data["Meta"] = map[string]string{"Title": "", "Description": ""}
	}
	if data["Flash"] == nil {
		data["Flash"] = nil
	}

	if h.render != nil {
		h.render.Render(w, r, tmpl, data)
		return
	}

	respond(w, http.StatusOK, data)
}

// HandleRegisterPage serves the registration form.
func (h *AuthHandler) HandleRegisterPage(w http.ResponseWriter, r *http.Request) {
	h.renderPage(w, r, "register.tmpl", map[string]any{
		"Meta": map[string]string{
			"Title": "Crear Cuenta",
		},
	})
}

// HandleRegister processes a new user registration via email + password.
func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderPage(w, r, "register.tmpl", map[string]any{
			"Error": "Datos de formulario inválidos",
			"Meta":  map[string]string{"Title": "Crear Cuenta"},
		})
		return
	}

	email    := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")
	fullName := strings.TrimSpace(r.FormValue("full_name"))

	formData := map[string]string{
		"Email":    email,
		"FullName": fullName,
	}

	if email == "" || password == "" || fullName == "" {
		h.renderPage(w, r, "register.tmpl", map[string]any{
			"Error":    "Nombre, correo y contraseña son obligatorios",
			"FormData": formData,
			"Meta":     map[string]string{"Title": "Crear Cuenta"},
		})
		return
	}

	if !strings.Contains(email, "@") {
		h.renderPage(w, r, "register.tmpl", map[string]any{
			"Error":    "Correo electrónico inválido",
			"FormData": formData,
			"Meta":     map[string]string{"Title": "Crear Cuenta"},
		})
		return
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		errMsg := "Error al crear la cuenta"
		if errors.Is(err, auth.ErrPasswordTooShort) {
			errMsg = "La contraseña debe tener al menos 12 caracteres"
		} else if errors.Is(err, auth.ErrPasswordTooLong) {
			errMsg = "La contraseña debe tener 72 caracteres o menos"
		} else {
			slog.Error("failed to hash password during registration", "error", err)
		}

		h.renderPage(w, r, "register.tmpl", map[string]any{
			"Error":    errMsg,
			"FormData": formData,
			"Meta":     map[string]string{"Title": "Crear Cuenta"},
		})
		return
	}

	user, err := h.users.Create(r.Context(), email, passwordHash, fullName)
	if err != nil {
		errMsg := "Error al crear la cuenta"
		if errors.Is(err, repo.ErrEmailTaken) {
			errMsg = "Ya existe una cuenta con ese correo electrónico"
		} else {
			slog.Error("failed to create user", "email", email, "error", err)
		}

		h.renderPage(w, r, "register.tmpl", map[string]any{
			"Error":    errMsg,
			"FormData": formData,
			"Meta":     map[string]string{"Title": "Crear Cuenta"},
		})
		return
	}

	_, err = h.sessions.Create(r.Context(), w, user.ID)
	if err != nil {
		slog.Error("failed to create session after registration",
			"user_id", user.ID, "error", err,
		)
		h.renderPage(w, r, "register.tmpl", map[string]any{
			"Error": "Error al crear la cuenta",
			"Meta":  map[string]string{"Title": "Crear Cuenta"},
		})
		return
	}

	slog.Info("user registered", "user_id", user.ID, "email", user.Email, "provider", "email")

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// HandleLoginPage serves the login form.
func (h *AuthHandler) HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	h.renderPage(w, r, "login.tmpl", map[string]any{
		"Meta": map[string]string{
			"Title": "Ingresar",
		},
	})
}

// HandleLogin processes a login attempt via email + password.
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderPage(w, r, "login.tmpl", map[string]any{
			"Error": "Datos de formulario inválidos",
			"Meta":  map[string]string{"Title": "Ingresar"},
		})
		return
	}

	email    := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	formData := map[string]string{
		"Email": email,
	}

	if email == "" || password == "" {
		h.renderPage(w, r, "login.tmpl", map[string]any{
			"Error":    "Correo y contraseña son obligatorios",
			"FormData": formData,
			"Meta":     map[string]string{"Title": "Ingresar"},
		})
		return
	}

	user, err := h.users.GetByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			h.renderPage(w, r, "login.tmpl", map[string]any{
				"Error":    "Correo o contraseña incorrectos",
				"FormData": formData,
				"Meta":     map[string]string{"Title": "Ingresar"},
			})
			return
		}
		slog.Error("failed to fetch user during login", "email", email, "error", err)
		h.renderPage(w, r, "login.tmpl", map[string]any{
			"Error": "Error al iniciar sesión",
			"Meta":  map[string]string{"Title": "Ingresar"},
		})
		return
	}

	if !user.HasPassword() {
		slog.Info("email login attempt on passwordless account", "email", email)
		h.renderPage(w, r, "login.tmpl", map[string]any{
			"Error":    "Correo o contraseña incorrectos",
			"FormData": formData,
			"Meta":     map[string]string{"Title": "Ingresar"},
		})
		return
	}

	if err := auth.VerifyPassword(password, *user.PasswordHash); err != nil {
		slog.Info("failed login attempt", "email", email)
		h.renderPage(w, r, "login.tmpl", map[string]any{
			"Error":    "Correo o contraseña incorrectos",
			"FormData": formData,
			"Meta":     map[string]string{"Title": "Ingresar"},
		})
		return
	}

	_, err = h.sessions.Create(r.Context(), w, user.ID)
	if err != nil {
		slog.Error("failed to create session after login",
			"user_id", user.ID, "error", err,
		)
		h.renderPage(w, r, "login.tmpl", map[string]any{
			"Error": "Error al iniciar sesión",
			"Meta":  map[string]string{"Title": "Ingresar"},
		})
		return
	}

	slog.Info("user logged in", "user_id", user.ID, "email", user.Email, "provider", "email")

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
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

	http.Redirect(w, r, "/", http.StatusSeeOther)
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
