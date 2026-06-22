// internal/handlers/profile_handler.go
// Profile view and edit endpoints.
// Rule 42: handlers = HTTP only. No SQL. No business logic.

package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"vallescentrales/internal/middleware"
	"vallescentrales/internal/repo"
)

// ProfileHandler handles profile view and edit endpoints.
type ProfileHandler struct {
	users  *repo.UserRepo
	render Renderer
}

// NewProfileHandler creates a ProfileHandler.
func NewProfileHandler(users *repo.UserRepo, render Renderer) *ProfileHandler {
	return &ProfileHandler{
		users:  users,
		render: render,
	}
}

// pageData builds standard template data for profile pages.
func (h *ProfileHandler) pageData(r *http.Request, title string, extra map[string]any) map[string]any {
	data := map[string]any{
		"Meta": map[string]string{
			"Title":       title,
			"Description": "Valles Centrales — Tu perfil",
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

// HandleProfileEditPage serves the profile edit form.
func (h *ProfileHandler) HandleProfileEditPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	h.render.Render(w, r, "profile_edit.tmpl", h.pageData(r, "Editar Perfil", map[string]any{
		"Profile": user,
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
		h.render.Render(w, r, "profile_edit.tmpl", h.pageData(r, "Editar Perfil", map[string]any{
			"Profile": user,
			"Error":   "Datos de formulario inválidos",
		}))
		return
	}

	// Parse all profile fields
	fullName     := strings.TrimSpace(r.FormValue("full_name"))
	displayName  := nullableString(strings.TrimSpace(r.FormValue("display_name")))
	username     := nullableString(strings.ToLower(strings.TrimSpace(r.FormValue("username"))))
	bio          := nullableString(strings.TrimSpace(r.FormValue("bio")))
	website      := nullableString(strings.TrimSpace(r.FormValue("website")))
	location     := nullableString(strings.TrimSpace(r.FormValue("location")))
	agencyName   := nullableString(strings.TrimSpace(r.FormValue("agency_name")))
	phone        := nullableString(strings.TrimSpace(r.FormValue("phone")))
	whatsapp     := nullableString(strings.TrimSpace(r.FormValue("whatsapp")))
	preferredLang := r.FormValue("preferred_lang")
	if preferredLang == "" {
		preferredLang = "es"
	}

	// Parse languages checkboxes
	languages := r.Form["languages"]
	if len(languages) == 0 {
		languages = []string{"es"}
	}

	// Parse privacy toggles
	showPhone    := r.FormValue("show_phone") == "on"
	showWhatsApp := r.FormValue("show_whatsapp") == "on"
	notifyEmail  := r.FormValue("notify_email") == "on"

	// Validate required fields
	if fullName == "" {
		h.render.Render(w, r, "profile_edit.tmpl", h.pageData(r, "Editar Perfil", map[string]any{
			"Profile": user,
			"Error":   "El nombre completo es obligatorio",
		}))
		return
	}

	// Validate username format if provided
	if username != nil {
		if err := validateUsername(*username); err != nil {
			h.render.Render(w, r, "profile_edit.tmpl", h.pageData(r, "Editar Perfil", map[string]any{
				"Profile": user,
				"Error":   err.Error(),
			}))
			return
		}
	}

	// Validate bio length
	if bio != nil && len([]rune(*bio)) > 160 {
		h.render.Render(w, r, "profile_edit.tmpl", h.pageData(r, "Editar Perfil", map[string]any{
			"Profile": user,
			"Error":   "La biografía no puede superar 160 caracteres",
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
		h.render.Render(w, r, "profile_edit.tmpl", h.pageData(r, "Editar Perfil", map[string]any{
			"Profile": user,
			"Error":   errMsg,
		}))
		return
	}

	slog.Info("profile updated", "user_id", updatedUser.ID)

	h.render.Render(w, r, "profile_edit.tmpl", h.pageData(r, "Editar Perfil", map[string]any{
		"Profile":  updatedUser,
		"Success":  "Perfil actualizado correctamente",
	}))
}

// HandlePublicProfile serves a user's public profile page.
func (h *ProfileHandler) HandlePublicProfile(w http.ResponseWriter, r *http.Request) {
	// TODO: implement after username routing is wired
	http.NotFound(w, r)
}

// nullableString returns nil if s is empty, otherwise &s.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// validateUsername checks username format rules.
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
