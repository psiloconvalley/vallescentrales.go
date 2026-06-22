// internal/auth/session_test.go
// Tests for session ID generation and cookie behavior.
// Session security is critical — these tests must always pass.

package auth

import (
	"encoding/hex"
	"testing"
)

// ─── generateSessionID ───────────────────────────────────────────────────────

func TestGenerateSessionID_Length(t *testing.T) {
	id, err := generateSessionID()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// 32 bytes = 64 hex chars
	if len(id) != sessionIDLength*2 {
		t.Fatalf("expected session ID length %d, got %d", sessionIDLength*2, len(id))
	}
}

func TestGenerateSessionID_IsHex(t *testing.T) {
	id, err := generateSessionID()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if _, err := hex.DecodeString(id); err != nil {
		t.Fatalf("expected valid hex string, got error: %v", err)
	}
}

func TestGenerateSessionID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := generateSessionID()
		if err != nil {
			t.Fatalf("expected no error on iteration %d, got: %v", i, err)
		}
		if ids[id] {
			t.Fatalf("duplicate session ID generated on iteration %d", i)
		}
		ids[id] = true
	}
}

func TestGenerateSessionID_EntropyBits(t *testing.T) {
	id, err := generateSessionID()
	if err != nil {
		t.Fatal(err)
	}

	bytes, err := hex.DecodeString(id)
	if err != nil {
		t.Fatal(err)
	}

	// Must be exactly 32 bytes = 256 bits of entropy
	if len(bytes) != 32 {
		t.Fatalf("expected 32 bytes entropy, got %d", len(bytes))
	}
}
