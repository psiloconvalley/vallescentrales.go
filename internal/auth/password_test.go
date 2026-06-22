// internal/auth/password_test.go
// Tests for Argon2id password hashing and verification.
// These tests are security-critical — do not remove or skip.

package auth

import (
	"strings"
	"testing"
)

// ─── HashPassword ────────────────────────────────────────────────────────────

func TestHashPassword_ValidPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("expected Argon2id hash format, got: %s", hash[:20])
	}
}

func TestHashPassword_TooShort(t *testing.T) {
	_, err := HashPassword("short")
	if err == nil {
		t.Fatal("expected error for password under 12 chars, got nil")
	}
	if err != ErrPasswordTooShort {
		t.Fatalf("expected ErrPasswordTooShort, got: %v", err)
	}
}

func TestHashPassword_TooLong(t *testing.T) {
	long := strings.Repeat("a", 73)
	_, err := HashPassword(long)
	if err == nil {
		t.Fatal("expected error for password over 72 chars, got nil")
	}
	if err != ErrPasswordTooLong {
		t.Fatalf("expected ErrPasswordTooLong, got: %v", err)
	}
}

func TestHashPassword_ExactMinLength(t *testing.T) {
	// 12 chars exactly — must succeed
	_, err := HashPassword("exactly12chr")
	if err != nil {
		t.Fatalf("expected no error for 12-char password, got: %v", err)
	}
}

func TestHashPassword_ExactMaxLength(t *testing.T) {
	// 72 chars exactly — must succeed
	max := strings.Repeat("a", 72)
	_, err := HashPassword(max)
	if err != nil {
		t.Fatalf("expected no error for 72-char password, got: %v", err)
	}
}

func TestHashPassword_UniqueHashes(t *testing.T) {
	// Same password must produce different hashes (salt is random)
	hash1, err := HashPassword("correct-horse-battery")
	if err != nil {
		t.Fatal(err)
	}
	hash2, err := HashPassword("correct-horse-battery")
	if err != nil {
		t.Fatal(err)
	}
	if hash1 == hash2 {
		t.Fatal("expected unique hashes for same password, got identical")
	}
}

// ─── VerifyPassword ──────────────────────────────────────────────────────────

func TestVerifyPassword_Correct(t *testing.T) {
	password := "correct-horse-battery"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}

	if err := VerifyPassword(password, hash); err != nil {
		t.Fatalf("expected nil for correct password, got: %v", err)
	}
}

func TestVerifyPassword_Wrong(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery")
	if err != nil {
		t.Fatal(err)
	}

	err = VerifyPassword("wrong-password-here!", hash)
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
	if err != ErrInvalidPassword {
		t.Fatalf("expected ErrInvalidPassword, got: %v", err)
	}
}

func TestVerifyPassword_EmptyPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery")
	if err != nil {
		t.Fatal(err)
	}

	err = VerifyPassword("", hash)
	if err == nil {
		t.Fatal("expected error for empty password, got nil")
	}
}

func TestVerifyPassword_MalformedHash(t *testing.T) {
	err := VerifyPassword("correct-horse-battery", "not-a-valid-hash")
	if err == nil {
		t.Fatal("expected error for malformed hash, got nil")
	}
	if err != ErrInvalidHash {
		t.Fatalf("expected ErrInvalidHash, got: %v", err)
	}
}

func TestVerifyPassword_TooShortPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery")
	if err != nil {
		t.Fatal(err)
	}

	// Short password — should return ErrInvalidPassword not panic
	err = VerifyPassword("short", hash)
	if err == nil {
		t.Fatal("expected error for short password, got nil")
	}
}

func TestVerifyPassword_TooLongPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery")
	if err != nil {
		t.Fatal(err)
	}

	long := strings.Repeat("a", 73)
	err = VerifyPassword(long, hash)
	if err == nil {
		t.Fatal("expected error for too-long password, got nil")
	}
}

// ─── Argon2id Format ─────────────────────────────────────────────────────────

func TestHashPassword_ContainsExpectedParams(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery")
	if err != nil {
		t.Fatal(err)
	}

	// Must contain OWASP-recommended parameters
	if !strings.Contains(hash, "m=65536") {
		t.Error("expected m=65536 (64MB memory) in hash")
	}
	if !strings.Contains(hash, "t=3") {
		t.Error("expected t=3 (3 iterations) in hash")
	}
	if !strings.Contains(hash, "p=4") {
		t.Error("expected p=4 (4 parallelism) in hash")
	}
}

// ─── Benchmark ───────────────────────────────────────────────────────────────

func BenchmarkHashPassword(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := HashPassword("correct-horse-battery")
		if err != nil {
			b.Fatal(err)
		}
	}
}
