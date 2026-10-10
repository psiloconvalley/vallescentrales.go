package models

import (
	"testing"
)

func strPtr(s string) *string {
	return &s
}

func TestUser_IsAdmin(t *testing.T) {
	admin := &User{Role: RoleAdmin}
	if !admin.IsAdmin() {
		t.Errorf("expected admin.IsAdmin() to be true, got false")
	}

	regular := &User{Role: RoleUser}
	if regular.IsAdmin() {
		t.Errorf("expected regular.IsAdmin() to be false, got true")
	}
}

func TestUser_DisplayNameOrFull(t *testing.T) {
	u1 := &User{DisplayName: strPtr("María Propietaria"), FullName: "María López"}
	if got := u1.DisplayNameOrFull(); got != "María Propietaria" {
		t.Errorf("expected 'María Propietaria', got %q", got)
	}

	u2 := &User{DisplayName: nil, FullName: "Juan Pérez"}
	if got := u2.DisplayNameOrFull(); got != "Juan Pérez" {
		t.Errorf("expected 'Juan Pérez', got %q", got)
	}

	u3 := &User{DisplayName: nil, FullName: ""}
	if got := u3.DisplayNameOrFull(); got != "" {
		t.Errorf("expected '', got %q", got)
	}
}

func TestUser_InitialLetter(t *testing.T) {
	u1 := User{FullName: "carlos"}
	if got := u1.InitialLetter(); got != "C" {
		t.Errorf("expected 'C', got %q", got)
	}

	u2 := User{FullName: "", Email: "ernesto@vallescentrales.com"}
	if got := u2.InitialLetter(); got != "E" {
		t.Errorf("expected 'E', got %q", got)
	}

	u3 := User{}
	if got := u3.InitialLetter(); got != "U" {
		t.Errorf("expected 'U', got %q", got)
	}
}

func TestUser_InitialLetter_Unicode(t *testing.T) {
	u := User{FullName: "álvaro"}
	if got := u.InitialLetter(); got != "Á" {
		t.Errorf("expected 'Á', got %q", got)
	}
}

func TestUser_HasPassword(t *testing.T) {
	u1 := &User{PasswordHash: strPtr("$argon2id$v=19$m=65536,t=3,p=2$...")}
	if !u1.HasPassword() {
		t.Errorf("expected u1.HasPassword() to be true, got false")
	}

	u2 := &User{PasswordHash: nil}
	if u2.HasPassword() {
		t.Errorf("expected u2.HasPassword() to be false, got true")
	}
}
