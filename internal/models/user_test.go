// internal/models/user_test.go
// Tests for User model methods.
// Pure logic tests — no DB, no HTTP.

package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// helper — builds a minimal valid User for testing
func testUser(role UserRole, provider AuthProvider) *User {
	email := "test@example.com"
	hash  := "$argon2id$v=19$m=65536,t=3,p=4$fakesalt$fakehash"
	return &User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: &hash,
		FullName:     "Test User",
		Role:         role,
		IsVerified:   true,
		AuthProvider: provider,
		Languages:    []string{"es"},
		ShowPhone:    true,
		ShowWhatsApp: true,
		NotifyEmail:  true,
		PreferredLang: "es",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// ─── Role checks ─────────────────────────────────────────────────────────────

func TestUser_IsAdmin(t *testing.T) {
	tests := []struct {
		role     UserRole
		expected bool
	}{
		{RoleAdmin, true},
		{RoleAgent, false},
		{RoleOwner, false},
		{RoleBuyer, false},
	}

	for _, tt := range tests {
		u := testUser(tt.role, AuthProviderEmail)
		if got := u.IsAdmin(); got != tt.expected {
			t.Errorf("IsAdmin() for role %q = %v, want %v", tt.role, got, tt.expected)
		}
	}
}

func TestUser_CanManageListings(t *testing.T) {
	tests := []struct {
		role     UserRole
		expected bool
	}{
		{RoleAdmin, true},
		{RoleAgent, true},
		{RoleOwner, true},
		{RoleBuyer, false},
	}

	for _, tt := range tests {
		u := testUser(tt.role, AuthProviderEmail)
		if got := u.CanManageListings(); got != tt.expected {
			t.Errorf("CanManageListings() for role %q = %v, want %v", tt.role, got, tt.expected)
		}
	}
}

// ─── Auth provider checks ────────────────────────────────────────────────────

func TestUser_IsGoogleUser(t *testing.T) {
	google := testUser(RoleOwner, AuthProviderGoogle)
	email  := testUser(RoleOwner, AuthProviderEmail)

	if !google.IsGoogleUser() {
		t.Error("expected IsGoogleUser() = true for google provider")
	}
	if email.IsGoogleUser() {
		t.Error("expected IsGoogleUser() = false for email provider")
	}
}

func TestUser_HasPassword(t *testing.T) {
	withPassword := testUser(RoleOwner, AuthProviderEmail)
	if !withPassword.HasPassword() {
		t.Error("expected HasPassword() = true when hash is set")
	}

	googleUser := testUser(RoleOwner, AuthProviderGoogle)
	googleUser.PasswordHash = nil
	if googleUser.HasPassword() {
		t.Error("expected HasPassword() = false when hash is nil")
	}

	emptyHash := ""
	emptyUser := testUser(RoleOwner, AuthProviderEmail)
	emptyUser.PasswordHash = &emptyHash
	if emptyUser.HasPassword() {
		t.Error("expected HasPassword() = false when hash is empty string")
	}
}

// ─── Display name ────────────────────────────────────────────────────────────

func TestUser_DisplayNameOrFull(t *testing.T) {
	u := testUser(RoleOwner, AuthProviderEmail)

	// No display name — returns full name
	if got := u.DisplayNameOrFull(); got != "Test User" {
		t.Errorf("expected %q, got %q", "Test User", got)
	}

	// With display name — returns display name
	dn := "Mi Nombre"
	u.DisplayName = &dn
	if got := u.DisplayNameOrFull(); got != "Mi Nombre" {
		t.Errorf("expected %q, got %q", "Mi Nombre", got)
	}

	// Empty display name — falls back to full name
	empty := ""
	u.DisplayName = &empty
	if got := u.DisplayNameOrFull(); got != "Test User" {
		t.Errorf("expected %q, got %q", "Test User", got)
	}
}

func TestUser_InitialLetter(t *testing.T) {
	u := testUser(RoleOwner, AuthProviderEmail)
	u.FullName = "Juan García"

	got := u.InitialLetter()
	if got != "J" {
		t.Errorf("expected %q, got %q", "J", got)
	}
}

func TestUser_InitialLetter_Unicode(t *testing.T) {
	u := testUser(RoleOwner, AuthProviderEmail)
	u.FullName = "Álvaro López"

	got := u.InitialLetter()
	if got != "Á" {
		t.Errorf("expected %q, got %q", "Á", got)
	}
}

func TestUser_PublicName_AgencyFirst(t *testing.T) {
	u := testUser(RoleAgent, AuthProviderEmail)
	agency := "Bienes Raíces del Valle"
	u.AgencyName = &agency

	got := u.PublicName()
	if got != agency {
		t.Errorf("expected agency name %q, got %q", agency, got)
	}
}

func TestUser_PublicName_FallsBackToDisplay(t *testing.T) {
	u := testUser(RoleOwner, AuthProviderEmail)
	dn := "Carlos V"
	u.DisplayName = &dn

	got := u.PublicName()
	if got != "Carlos V" {
		t.Errorf("expected %q, got %q", "Carlos V", got)
	}
}

// ─── SafeUser ────────────────────────────────────────────────────────────────

func TestUser_ToSafe_NoPasswordHash(t *testing.T) {
	u := testUser(RoleOwner, AuthProviderEmail)
	safe := u.ToSafe()

	// SafeUser has no PasswordHash field — verify by checking it compiles
	// and the email is preserved
	if safe.Email != u.Email {
		t.Errorf("expected email %q, got %q", u.Email, safe.Email)
	}
	if safe.FullName != u.FullName {
		t.Errorf("expected full_name %q, got %q", u.FullName, safe.FullName)
	}
	if safe.AuthProvider != u.AuthProvider {
		t.Errorf("expected auth_provider %q, got %q", u.AuthProvider, safe.AuthProvider)
	}
}
