package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAgent UserRole = "agent"
	RoleAdmin UserRole = "admin"
)

type AuthProvider string

const (
	AuthProviderLocal   AuthProvider = "local"
	AuthProviderGoogle  AuthProvider = "google"
	AuthProviderPasskey AuthProvider = "passkey"
)

type User struct {
	ID                  uuid.UUID    `json:"id"`
	Email               string       `json:"email"`
	PasswordHash        *string      `json:"-"`
	FullName            string       `json:"full_name"`
	DisplayName         *string      `json:"display_name,omitempty"`
	Username            *string      `json:"username,omitempty"`
	Phone               *string      `json:"phone,omitempty"`
	WhatsApp            *string      `json:"whatsapp,omitempty"`
	ShowPhone           bool         `json:"show_phone"`
	ShowWhatsApp        bool         `json:"show_whatsapp"`
	Role                UserRole     `json:"role"`
	UserType            string       `json:"user_type"` // particular, agente, inmobiliaria, admin
	IsVerified          bool         `json:"is_verified"`
	GoogleID            *string      `json:"google_id,omitempty"`
	AuthProvider        AuthProvider `json:"auth_provider"`
	AvatarURL           *string      `json:"avatar_url,omitempty"`
	Bio                 *string      `json:"bio,omitempty"`
	Website             *string      `json:"website,omitempty"`
	Location            *string      `json:"location,omitempty"`
	AgencyName          *string      `json:"agency_name,omitempty"`
	Languages           []string     `json:"languages,omitempty"`
	Municipality        *string      `json:"municipality,omitempty"`
	NotifyEmail         bool         `json:"notify_email"`
	PreferredLang       string       `json:"preferred_lang"`
	OnboardingCompleted bool         `json:"onboarding_completed"`
	CreatedAt           time.Time    `json:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
}

// DisplayNameOrFull returns DisplayName if set, otherwise FullName.
func (u *User) DisplayNameOrFull() string {
	if u.DisplayName != nil && *u.DisplayName != "" {
		return *u.DisplayName
	}
	return u.FullName
}

// Initial returns the first letter of the display name for avatar fallbacks.
func (u *User) Initial() string {
	name := u.DisplayNameOrFull()
	if len(name) == 0 {
		return "?"
	}
	return string([]rune(name)[0])
}

// HasPassword returns true if the user has a password set.
func (u *User) HasPassword() bool {
	return u.PasswordHash != nil && *u.PasswordHash != ""
}

// CanManageListings returns true if the user has permission to manage listings.
func (u *User) CanManageListings() bool {
	return true
}

// IsAdmin returns true if the user has an admin role.
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// InitialLetter returns the uppercase first character of the user's name or email.
func (u User) InitialLetter() string {
	if u.FullName != "" {
		runes := []rune(u.FullName)
		if len(runes) > 0 {
			return strings.ToUpper(string(runes[0]))
		}
	}
	if u.Email != "" {
		runes := []rune(u.Email)
		if len(runes) > 0 {
			return strings.ToUpper(string(runes[0]))
		}
	}
	return "U"
}
