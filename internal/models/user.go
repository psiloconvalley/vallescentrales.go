// internal/models/user.go
// Pure Go types for the users table.
// Derived from migrations/000001_create_users.up.sql
// Derived from migrations/000005_add_google_auth.up.sql
// No DB code. No HTTP code. No business logic.

package models

import (
	"time"

	"github.com/google/uuid"
)

// UserRole maps exactly to the user_role ENUM in PostgreSQL.
// Canonical vocabulary — ADR-005. Never use raw strings.
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
	ID           uuid.UUID    `db:"id"            json:"id"`
	Email        string       `db:"email"         json:"email"`
	PasswordHash *string      `db:"password_hash" json:"-"` // nullable — Google users have no password
	FullName     string       `db:"full_name"     json:"full_name"`
	Phone        *string      `db:"phone"         json:"phone,omitempty"`
	WhatsApp     *string      `db:"whatsapp"      json:"whatsapp,omitempty"`
	Role         UserRole     `db:"role"          json:"role"`
	IsVerified   bool         `db:"is_verified"   json:"is_verified"`
	GoogleID     *string      `db:"google_id"     json:"-"` // never expose
	AuthProvider AuthProvider `db:"auth_provider" json:"auth_provider"`
	CreatedAt    time.Time    `db:"created_at"    json:"created_at"`
	UpdatedAt    time.Time    `db:"updated_at"    json:"updated_at"`
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
}

// ToSafe strips sensitive fields before sending to client.
func (u *User) ToSafe() SafeUser {
	return SafeUser{
		ID:           u.ID,
		Email:        u.Email,
		FullName:     u.FullName,
		Phone:        u.Phone,
		WhatsApp:     u.WhatsApp,
		Role:         u.Role,
		IsVerified:   u.IsVerified,
		AuthProvider: u.AuthProvider,
		CreatedAt:    u.CreatedAt,
	}
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
// Google-only users do not have a password.
func (u *User) HasPassword() bool {
	return u.PasswordHash != nil && *u.PasswordHash != ""
}

// IsGoogleUser returns true if the user registered via Google OAuth.
func (u *User) IsGoogleUser() bool {
	return u.AuthProvider == AuthProviderGoogle
}
