// internal/services/storage_test.go
// Tests for storage key generation and content-type validation.
// No live R2 access needed — pure logic tests.

package services

import (
	"strings"
	"testing"
)

// ─── AllowedImageType ────────────────────────────────────────────────────────

func TestAllowedImageType(t *testing.T) {
	tests := []struct {
		contentType string
		allowed     bool
	}{
		{"image/jpeg", true},
		{"image/jpg", true},
		{"image/png", true},
		{"image/webp", true},
		{"image/gif", false},
		{"application/pdf", false},
		{"text/plain", false},
		{"", false},
	}

	for _, tt := range tests {
		got := AllowedImageType(tt.contentType)
		if got != tt.allowed {
			t.Errorf("AllowedImageType(%q) = %v, want %v", tt.contentType, got, tt.allowed)
		}
	}
}

// ─── sanitizeFilename ────────────────────────────────────────────────────────

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"My House Photo", "my-house-photo"},
		{"Terreno_123", "terreno_123"},
		{"Casa!! Bonita??", "casa-bonita"},
		{"Rancho Del Valle", "rancho-del-valle"},
		{"123 Main St.", "123-main-st"},
		{"áéíóú", ""}, // non-ascii stripped
		{"", ""},
	}

	for _, tt := range tests {
		got := sanitizeFilename(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// ─── generateStorageKey ──────────────────────────────────────────────────────

func TestGenerateStorageKey_HasFolderPrefix(t *testing.T) {
	key := generateStorageKey("listings", "my-house.jpg")

	if !strings.HasPrefix(key, "listings/") {
		t.Fatalf("expected key to start with listings/, got: %s", key)
	}
}

func TestGenerateStorageKey_PreservesExtension(t *testing.T) {
	tests := []struct {
		filename string
		ext      string
	}{
		{"house.jpg", ".jpg"},
		{"lot.png", ".png"},
		{"cabin.webp", ".webp"},
		{"photo.jpeg", ".jpeg"},
	}

	for _, tt := range tests {
		key := generateStorageKey("avatars", tt.filename)
		if !strings.HasSuffix(key, tt.ext) {
			t.Errorf("expected key %q to end with %q", key, tt.ext)
		}
	}
}

func TestGenerateStorageKey_DefaultsToBin(t *testing.T) {
	key := generateStorageKey("misc", "filewithoutdot")
	if !strings.HasSuffix(key, ".bin") {
		t.Fatalf("expected .bin extension, got: %s", key)
	}
}

func TestGenerateStorageKey_IncludesSanitizedBaseName(t *testing.T) {
	key := generateStorageKey("listings", "Casa Bonita!!.jpg")
	if !strings.Contains(key, "casa-bonita") {
		t.Fatalf("expected sanitized basename in key, got: %s", key)
	}
}

func TestGenerateStorageKey_Unique(t *testing.T) {
	key1 := generateStorageKey("listings", "photo.jpg")
	key2 := generateStorageKey("listings", "photo.jpg")

	if key1 == key2 {
		t.Fatalf("expected unique storage keys, got duplicates: %s", key1)
	}
}

func TestGenerateStorageKey_LimitsBaseNameLength(t *testing.T) {
	longName := strings.Repeat("a", 100) + ".jpg"
	key := generateStorageKey("listings", longName)

	parts := strings.Split(key, "-")
	if len(parts) < 3 {
		t.Fatalf("unexpected key format: %s", key)
	}

	// Last segment before extension should be truncated
	if len(key) > 100 {
		t.Fatalf("expected reasonable key length, got: %d", len(key))
	}
}

// ─── PublicURL ───────────────────────────────────────────────────────────────

func TestPublicURL(t *testing.T) {
	s := &StorageService{
		publicURL: "https://pub-example.r2.dev",
	}

	got := s.PublicURL("listings/20250622-abc-house.jpg")
	want := "https://pub-example.r2.dev/listings/20250622-abc-house.jpg"

	if got != want {
		t.Errorf("PublicURL() = %q, want %q", got, want)
	}
}
