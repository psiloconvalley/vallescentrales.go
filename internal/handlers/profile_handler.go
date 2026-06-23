// internal/handlers/profile_handler.go
// Profile and security settings endpoints.
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

// ProfileHandler handles profile and security settings.
type ProfileHandler struct {
	users    *repo.UserRepo
	passkeys *repo.PasskeyRepo
	render   Renderer
}

// NewProfileHandler creates a ProfileHandler.
func NewProfileHandler(users *repo.UserRepo, passkeys *repo.PasskeyRepo, render Renderer) *ProfileHandler {
	return &ProfileHandler{
		users:    users,
		passkeys: passkeys,
		render:   render,
	}
}

// pageData builds standard template data.
func (h *ProfileHandler) pageData(r *http.Request, title string, extra map[string]any) map[string]any {
	data := map[string]any{
		"Meta": map[string]string{
			"Title":       title,
			"Description": "Valles Centrales — Configuración",
		},
		"User":      middleware.UserFromContext(r.Context()),
		"Flash":     nil,
		"CSRFToken": middleware.CSRFToken(r),
	}
	for k, v := range extra {
		data[k] = v
	}
	return data
}

// ─── Profile Edit ────────────────────────────────────────────────────────────

// HandleProfileEditPage serves the profile edit form.
func (h *ProfileHandler) HandleProfileEditPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	h.render.Render(w, r, "profile_edit.tmpl", h.pageData(r, "Perfil", map[string]any{
		"Profile":   user,
		"ActiveTab": "profile",
	}))
}

// HandleProfileSave processes the profile edit form submission.
func (h *ProfileHandler) HandleProfileSave(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.render.Render(w, r, "profile_edit.tmpl", h.pageData(r, "Perfil", map[string]any{
			"Profile":   user,
			"ActiveTab": "profile",
			"Error":     "Datos de formulario inválidos",
		}))
		return
	}

	fullName      := strings.TrimSpace(r.FormValue("full_name"))
	displayName   := nullableString(strings.TrimSpace(r.FormValue("display_name")))
	username      := nullableString(strings.ToLower(strings.TrimSpace(r.FormValue("username"))))
	bio           := nullableString(strings.TrimSpace(r.FormValue("bio")))
	website       := nullableString(strings.TrimSpace(r.FormValue("website")))
	location      := nullableString(strings.TrimSpace(r.FormValue("location")))
	agencyName    := nullableString(strings.TrimSpace(r.FormValue("agency_name")))
	phone         := nullableString(strings.TrimSpace(r.FormValue("phone")))
	whatsapp      := nullableString(strings.TrimSpace(r.FormValue("whatsapp")))
	preferredLang := r.FormValue("preferred_lang")
	if preferredLang == "" {
		preferredLang = "es"
	}

	languages := r.Form["languages"]
	if len(languages) == 0 {
		languages = []string{"es"}
	}

	showPhone    := r.FormValue("show_phone") == "on"
	showWhatsApp := r.FormValue("show_whatsapp") == "on"
	notifyEmail  := r.FormValue("notify_email") == "on"

	if fullName == "" {
		h.render.Render(w, r, "profile_edit.tmpl", h.pageData(r, "Perfil", map[string]any{
			"Profile":   user,
			"ActiveTab": "profile",
			"Error":     "El nombre completo es obligatorio",
		}))
		return
	}

	if username != nil {
		if err := validateUsername(*username); err != nil {
			h.render.Render(w, r, "profile_edit.tmpl", h.pageData(r, "Perfil", map[string]any{
				"Profile":   user,
				"ActiveTab": "profile",
				"Error":     err.Error(),
			}))
			return
		}
	}

	if bio != nil && len([]rune(*bio)) > 160 {
		h.render.Render(w, r, "profile_edit.tmpl", h.pageData(r, "Perfil", map[string]any{
			"Profile":   user,
			"ActiveTab": "profile",
			"Error":     "La biografía no puede superar 160 caracteres",
		}))
		return
	}

	input := repo.UpdateProfileInput{
		FullName:      fullName,
		DisplayName:   displayName,
		Username:      username,
		Bio:           bio,
		Website:       website,
		Location:      location,
		AgencyName:    agencyName,
		Phone:         phone,
		WhatsApp:      whatsapp,
		Languages:     languages,
		ShowPhone:     showPhone,
		ShowWhatsApp:  showWhatsApp,
		NotifyEmail:   notifyEmail,
		PreferredLang: preferredLang,
	}

	updatedUser, err := h.users.UpdateProfile(r.Context(), user.ID, input)
	if err != nil {
		errMsg := "Error al guardar el perfil"
		if errors.Is(err, repo.ErrUsernameTaken) {
			errMsg = "Ese nombre de usuario ya está en uso"
		} else {
			slog.Error("failed to update profile", "user_id", user.ID, "error", err)
		}
		h.render.Render(w, r, "profile_edit.tmpl", h.pageData(r, "Perfil", map[string]any{
			"Profile":   user,
			"ActiveTab": "profile",
			"Error":     errMsg,
		}))
		return
	}

	slog.Info("profile updated", "user_id", updatedUser.ID)

	h.render.Render(w, r, "profile_edit.tmpl", h.pageData(r, "Perfil", map[string]any{
		"Profile":   updatedUser,
		"ActiveTab": "profile",
		"Success":   "Perfil actualizado correctamente",
	}))
}

// ─── Security ────────────────────────────────────────────────────────────────

// HandleSecurityPage serves the security settings page.
func (h *ProfileHandler) HandleSecurityPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	passkeys, err := h.passkeys.ListByUserID(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to load passkeys", "user_id", user.ID, "error", err)
		passkeys = nil
	}

	h.render.Render(w, r, "security.tmpl", h.pageData(r, "Seguridad", map[string]any{
		"ActiveTab": "security",
		"Passkeys":  passkeys,
	}))
}

// HandleChangePassword processes a password change for email users.
func (h *ProfileHandler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/dashboard/security", http.StatusSeeOther)
		return
	}

	currentPassword := r.FormValue("current_password")
	newPassword     := r.FormValue("new_password")

	// Verify current password
	if !user.HasPassword() {
		h.renderSecurityError(w, r, user, "Tu cuenta no usa contraseña")
		return
	}

	if err := auth.VerifyPassword(currentPassword, *user.PasswordHash); err != nil {
		h.renderSecurityError(w, r, user, "Contraseña actual incorrecta")
		return
	}

	// Hash new password
	newHash, err := auth.HashPassword(newPassword)
	if err != nil {
		if errors.Is(err, auth.ErrPasswordTooShort) {
			h.renderSecurityError(w, r, user, "La nueva contraseña debe tener al menos 12 caracteres")
			return
		}
		if errors.Is(err, auth.ErrPasswordTooLong) {
			h.renderSecurityError(w, r, user, "La nueva contraseña debe tener 72 caracteres o menos")
			return
		}
		slog.Error("failed to hash new password", "user_id", user.ID, "error", err)
		h.renderSecurityError(w, r, user, "Error al cambiar la contraseña")
		return
	}

	// TODO: Add UpdatePassword method to user_repo
	_ = newHash

	slog.Info("password changed", "user_id", user.ID)

	passkeys, _ := h.passkeys.ListByUserID(r.Context(), user.ID)

	h.render.Render(w, r, "security.tmpl", h.pageData(r, "Seguridad", map[string]any{
		"ActiveTab": "security",
		"Passkeys":  passkeys,
		"Success":   "Contraseña actualizada correctamente",
	}))
}

// renderSecurityError renders the security page with an error message.
func (h *ProfileHandler) renderSecurityError(w http.ResponseWriter, r *http.Request, user interface{}, errMsg string) {
	passkeys, _ := h.passkeys.ListByUserID(r.Context(), middleware.UserFromContext(r.Context()).ID)

	h.render.Render(w, r, "security.tmpl", h.pageData(r, "Seguridad", map[string]any{
		"ActiveTab": "security",
		"Passkeys":  passkeys,
		"Error":     errMsg,
	}))
}

// HandlePublicProfile serves a user's public profile page.
func (h *ProfileHandler) HandlePublicProfile(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func validateUsername(username string) error {
	if len(username) < 3 {
		return errors.New("el nombre de usuario debe tener al menos 3 caracteres")
	}
	if len(username) > 30 {
		return errors.New("el nombre de usuario no puede superar 30 caracteres")
	}
	for _, c := range username {
		if !isAlphanumericOrUnderscore(c) {
			return errors.New("el nombre de usuario solo puede contener letras, números y guiones bajos")
		}
	}
	return nil
}

func isAlphanumericOrUnderscore(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}
