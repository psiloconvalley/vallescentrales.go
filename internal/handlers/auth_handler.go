// internal/handlers/auth_handler.go

package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"vallescentrales/internal/auth"
	"vallescentrales/internal/middleware"
	"vallescentrales/internal/repo"
)

type AuthHandler struct {
	users      *repo.UserRepo
	sessions   *auth.SessionManager
	googleAuth *auth.GoogleOAuth
	render     Renderer
	production bool
}

func NewAuthHandler(
	users *repo.UserRepo,
	sessions *auth.SessionManager,
	googleAuth *auth.GoogleOAuth,
	render Renderer,
	production bool,
) *AuthHandler {
	return &AuthHandler{
		users:      users,
		sessions:   sessions,
		googleAuth: googleAuth,
		render:     render,
		production: production,
	}
}

func (h *AuthHandler) renderPage(w http.ResponseWriter, r *http.Request, name string, data map[string]any) {
	if data == nil {
		data = make(map[string]any)
	}
	data["GoogleEnabled"] = h.googleAuth.Enabled()
	data["CSRFToken"] = middleware.CSRFToken(r)
	data["User"] = middleware.UserFromContext(r.Context())
	h.render.Render(w, r, name, data)
}

func (h *AuthHandler) HandleRegisterPage(w http.ResponseWriter, r *http.Request) {
	if middleware.UserFromContext(r.Context()) != nil {
		http.Redirect(w, r, "/cuenta", http.StatusSeeOther)
		return
	}
	h.renderPage(w, r, "register.tmpl", map[string]any{
		"Meta": map[string]string{"Title": "Crear Cuenta"},
	})
}

func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderPage(w, r, "register.tmpl", map[string]any{
			"Error": "Datos de formulario inválidos",
			"Meta":  map[string]string{"Title": "Crear Cuenta"},
		})
		return
	}

	fullName := strings.TrimSpace(r.FormValue("full_name"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	formData := map[string]string{
		"FullName": fullName,
		"Email":    email,
	}

	if fullName == "" {
		h.renderPage(w, r, "register.tmpl", map[string]any{
			"Error":    "El nombre completo es obligatorio",
			"FormData": formData,
			"Meta":     map[string]string{"Title": "Crear Cuenta"},
		})
		return
	}

	if email == "" {
		h.renderPage(w, r, "register.tmpl", map[string]any{
			"Error":    "El correo electrónico es obligatorio",
			"FormData": formData,
			"Meta":     map[string]string{"Title": "Crear Cuenta"},
		})
		return
	}

	if len(password) < 12 {
		h.renderPage(w, r, "register.tmpl", map[string]any{
			"Error":    "La contraseña debe tener al menos 12 caracteres",
			"FormData": formData,
			"Meta":     map[string]string{"Title": "Crear Cuenta"},
		})
		return
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		h.renderPage(w, r, "register.tmpl", map[string]any{
			"Error":    "Error al procesar el registro",
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
			"Error": "Cuenta creada pero falló el inicio de sesión automático. Por favor ingresa.",
			"Meta":  map[string]string{"Title": "Crear Cuenta"},
		})
		return
	}

	slog.Info("user registered and logged in",
		"user_id", user.ID,
		"email", user.Email,
		"provider", "email",
	)

	http.Redirect(w, r, "/cuenta", http.StatusSeeOther)
}

func (h *AuthHandler) HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	if middleware.UserFromContext(r.Context()) != nil {
		http.Redirect(w, r, "/cuenta", http.StatusSeeOther)
		return
	}

	redirect := r.URL.Query().Get("redirect")
	if !isSafeRedirect(redirect) {
		redirect = ""
	}

	h.renderPage(w, r, "login.tmpl", map[string]any{
		"Redirect": redirect,
		"Meta":     map[string]string{"Title": "Ingresar"},
	})
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderPage(w, r, "login.tmpl", map[string]any{
			"Error": "Datos de formulario inválidos",
			"Meta":  map[string]string{"Title": "Ingresar"},
		})
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	redirect := r.FormValue("redirect")

	formData := map[string]string{
		"Email": email,
	}

	if email == "" || password == "" {
		h.renderPage(w, r, "login.tmpl", map[string]any{
			"Error":    "Introduce tu correo y contraseña",
			"FormData": formData,
			"Redirect": redirect,
			"Meta":     map[string]string{"Title": "Ingresar"},
		})
		return
	}

	user, err := h.users.GetByEmail(r.Context(), email)
	if err != nil {
		if !errors.Is(err, repo.ErrNotFound) {
			slog.Error("failed to lookup user by email", "email", email, "error", err)
		}
		h.renderPage(w, r, "login.tmpl", map[string]any{
			"Error":    "Correo o contraseña incorrectos",
			"FormData": formData,
			"Redirect": redirect,
			"Meta":     map[string]string{"Title": "Ingresar"},
		})
		return
	}

	if !user.HasPassword() {
		h.renderPage(w, r, "login.tmpl", map[string]any{
			"Error":    "Esta cuenta fue creada con Google. Inicia sesión con Google.",
			"FormData": formData,
			"Redirect": redirect,
			"Meta":     map[string]string{"Title": "Ingresar"},
		})
		return
	}

	if err := auth.VerifyPassword(password, *user.PasswordHash); err != nil {
		h.renderPage(w, r, "login.tmpl", map[string]any{
			"Error":    "Correo o contraseña incorrectos",
			"FormData": formData,
			"Redirect": redirect,
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
			"Error":    "Error al iniciar sesión",
			"Redirect": redirect,
			"Meta":     map[string]string{"Title": "Ingresar"},
		})
		return
	}

	slog.Info("user logged in", "user_id", user.ID, "email", user.Email, "provider", "email")

	target := "/cuenta"
	if isSafeRedirect(redirect) {
		target = redirect
	}

	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if err := h.sessions.Destroy(r.Context(), w, r); err != nil {
		slog.Error("failed to destroy session on logout", "error", err)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if !h.googleAuth.Enabled() {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	h.googleAuth.RedirectToGoogle(w, r)
}

func (h *AuthHandler) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if !h.googleAuth.Enabled() {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	googleUser, err := h.googleAuth.ProcessCallback(w, r)
	if err != nil {
		slog.Error("google oauth callback failed", "error", err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	ctx := r.Context()

	// 1. Existing Google user
	user, err := h.users.GetByGoogleID(ctx, googleUser.ID)
	if err == nil {
		_, err = h.sessions.Create(ctx, w, user.ID)
		if err != nil {
			slog.Error("failed to create session for google user", "user_id", user.ID, "error", err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		slog.Info("user logged in via google", "user_id", user.ID, "email", user.Email)
		http.Redirect(w, r, "/cuenta", http.StatusSeeOther)
		return
	}

	// 2. Existing email user — link accounts
	existingUser, err := h.users.GetByEmail(ctx, googleUser.Email)
	if err == nil {
		user, err = h.users.LinkGoogleAccount(ctx, existingUser.ID, googleUser.ID)
		if err != nil {
			slog.Error("failed to link google account", "user_id", existingUser.ID, "error", err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		_, err = h.sessions.Create(ctx, w, user.ID)
		if err != nil {
			slog.Error("failed to create session after google link", "user_id", user.ID, "error", err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		slog.Info("google account linked", "user_id", user.ID, "email", user.Email)
		http.Redirect(w, r, "/cuenta", http.StatusSeeOther)
		return
	}

	// 3. New user registration via Google
	user, err = h.users.CreateGoogle(ctx, googleUser.Email, googleUser.Name, googleUser.ID)
	if err != nil {
		slog.Error("failed to create google user", "email", googleUser.Email, "error", err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	_, err = h.sessions.Create(ctx, w, user.ID)
	if err != nil {
		slog.Error("failed to create session for new google user", "user_id", user.ID, "error", err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	slog.Info("user registered via google", "user_id", user.ID, "email", user.Email)
	http.Redirect(w, r, "/cuenta", http.StatusSeeOther)
}

func isSafeRedirect(target string) bool {
	if target == "" {
		return false
	}
	if !strings.HasPrefix(target, "/") {
		return false
	}
	if strings.HasPrefix(target, "//") {
		return false
	}
	u, err := url.Parse(target)
	if err != nil {
		return false
	}
	return u.Host == "" && u.Scheme == ""
}
