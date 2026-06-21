// internal/models/user.go
// Pure Go types for the users table.
// Derived from migrations/000001_create_users.up.sql
// Derived from migrations/000005_add_google_auth.up.sql
// Derived from migrations/000006_add_profile_fields.up.sql
// No DB code. No HTTP code. No business logic.

package models

import (
	"time"

	"github.com/google/uuid"
)

// UserRole maps exactly to the user_role ENUM in PostgreSQL.
type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleAgent UserRole = "agent"
	RoleOwner UserRole = "owner"
	RoleBuyer UserRole = "buyer"
)

// AuthProvider tracks how the user registered.
type AuthProvider string

const (
	AuthProviderEmail  AuthProvider = "email"
	AuthProviderGoogle AuthProvider = "google"
)

// User maps exactly to the users table.
// Column order matches the schema for readability.
type User struct {
	ID           uuid.UUID    `db:"id"             json:"id"`
	Email        string       `db:"email"          json:"email"`
	PasswordHash *string      `db:"password_hash"  json:"-"`
	FullName     string       `db:"full_name"      json:"full_name"`
	Phone        *string      `db:"phone"          json:"phone,omitempty"`
	WhatsApp     *string      `db:"whatsapp"       json:"whatsapp,omitempty"`
	Role         UserRole     `db:"role"           json:"role"`
	IsVerified   bool         `db:"is_verified"    json:"is_verified"`
	GoogleID     *string      `db:"google_id"      json:"-"`
	AuthProvider AuthProvider `db:"auth_provider"  json:"auth_provider"`
	CreatedAt    time.Time    `db:"created_at"     json:"created_at"`
	UpdatedAt    time.Time    `db:"updated_at"     json:"updated_at"`

	// Profile fields — migration 000006
	Username      *string  `db:"username"       json:"username,omitempty"`
	DisplayName   *string  `db:"display_name"   json:"display_name,omitempty"`
	Bio           *string  `db:"bio"            json:"bio,omitempty"`
	AvatarURL     *string  `db:"avatar_url"     json:"avatar_url,omitempty"`
	Website       *string  `db:"website"        json:"website,omitempty"`
	Location      *string  `db:"location"       json:"location,omitempty"`
	AgencyName    *string  `db:"agency_name"    json:"agency_name,omitempty"`
	Languages     []string `db:"languages"      json:"languages"`
	ShowPhone     bool     `db:"show_phone"     json:"show_phone"`
	ShowWhatsApp  bool     `db:"show_whatsapp"  json:"show_whatsapp"`
	NotifyEmail   bool     `db:"notify_email"   json:"notify_email"`
	PreferredLang string   `db:"preferred_lang" json:"preferred_lang"`
}

// SafeUser is what we send over the wire.
// PasswordHash and GoogleID are never included.
type SafeUser struct {
	ID           uuid.UUID    `json:"id"`
	Email        string       `json:"email"`
	FullName     string       `json:"full_name"`
	Phone        *string      `json:"phone,omitempty"`
	WhatsApp     *string      `json:"whatsapp,omitempty"`
	Role         UserRole     `json:"role"`
	IsVerified   bool         `json:"is_verified"`
	AuthProvider AuthProvider `json:"auth_provider"`
	CreatedAt    time.Time    `json:"created_at"`
	Username     *string      `json:"username,omitempty"`
	DisplayName  *string      `json:"display_name,omitempty"`
	Bio          *string      `json:"bio,omitempty"`
	AvatarURL    *string      `json:"avatar_url,omitempty"`
	Website      *string      `json:"website,omitempty"`
	Location     *string      `json:"location,omitempty"`
	AgencyName   *string      `json:"agency_name,omitempty"`
	Languages    []string     `json:"languages"`
	ShowPhone    bool         `json:"show_phone"`
	ShowWhatsApp bool         `json:"show_whatsapp"`
	PreferredLang string      `json:"preferred_lang"`
}

// ToSafe strips sensitive fields before sending to client.
func (u *User) ToSafe() SafeUser {
	return SafeUser{
		ID:            u.ID,
		Email:         u.Email,
		FullName:      u.FullName,
		Phone:         u.Phone,
		WhatsApp:      u.WhatsApp,
		Role:          u.Role,
		IsVerified:    u.IsVerified,
		AuthProvider:  u.AuthProvider,
		CreatedAt:     u.CreatedAt,
		Username:      u.Username,
		DisplayName:   u.DisplayName,
		Bio:           u.Bio,
		AvatarURL:     u.AvatarURL,
		Website:       u.Website,
		Location:      u.Location,
		AgencyName:    u.AgencyName,
		Languages:     u.Languages,
		ShowPhone:     u.ShowPhone,
		ShowWhatsApp:  u.ShowWhatsApp,
		PreferredLang: u.PreferredLang,
	}
}

// DisplayNameOrFull returns the display name if set, otherwise full name.
func (u *User) DisplayNameOrFull() string {
	if u.DisplayName != nil && *u.DisplayName != "" {
		return *u.DisplayName
	}
	return u.FullName
}

// PublicName returns the name to show publicly on listings.
func (u *User) PublicName() string {
	if u.AgencyName != nil && *u.AgencyName != "" {
		return *u.AgencyName
	}
	return u.DisplayNameOrFull()
}

// InitialLetter returns the first letter of the display name for avatars.
func (u *User) InitialLetter() string {
	name := u.DisplayNameOrFull()
	if len(name) == 0 {
		return "?"
	}
	return string([]rune(name)[0])
}

// CanManageListings returns true if the user can create and edit listings.
func (u *User) CanManageListings() bool {
	return u.Role == RoleAdmin || u.Role == RoleAgent || u.Role == RoleOwner
}

// IsAdmin returns true if the user has the admin role.
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// HasPassword returns true if the user has a password set.
func (u *User) HasPassword() bool {
	return u.PasswordHash != nil && *u.PasswordHash != ""
}

// IsGoogleUser returns true if the user registered via Google OAuth.
func (u *User) IsGoogleUser() bool {
	return u.AuthProvider == AuthProviderGoogle
}
