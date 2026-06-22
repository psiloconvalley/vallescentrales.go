// internal/models/listing_test.go
// Tests for Listing model methods.
// Pure logic tests — no DB, no HTTP.

package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// helper — builds a minimal valid Listing for testing
func testListing(status ListingStatus, propertyType PropertyType) *Listing {
	ownerID := uuid.New()
	price   := 1500000.00
	return &Listing{
		ID:           uuid.New(),
		OwnerID:      ownerID,
		Title:        "Casa en Tlacolula de Matamoros",
		Slug:         "casa-en-tlacolula-de-matamoros",
		PropertyType: propertyType,
		Status:       status,
		PriceMXN:     price,
		Municipality: "tlacolula_de_matamoros",
		IsFeatured:   false,
		ViewCount:    0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// ─── IsPublic ────────────────────────────────────────────────────────────────

func TestListing_IsPublic(t *testing.T) {
	tests := []struct {
		status   ListingStatus
		expected bool
	}{
		{StatusActive,        true},
		{StatusDraft,         false},
		{StatusUnderContract, false},
		{StatusSold,          false},
		{StatusArchived,      false},
	}

	for _, tt := range tests {
		l := testListing(tt.status, TypeHouse)
		if got := l.IsPublic(); got != tt.expected {
			t.Errorf("IsPublic() for status %q = %v, want %v", tt.status, got, tt.expected)
		}
	}
}

// ─── IsOwnedBy ───────────────────────────────────────────────────────────────

func TestListing_IsOwnedBy(t *testing.T) {
	l := testListing(StatusActive, TypeHouse)

	if !l.IsOwnedBy(l.OwnerID) {
		t.Error("expected IsOwnedBy() = true for listing owner")
	}

	other := uuid.New()
	if l.IsOwnedBy(other) {
		t.Error("expected IsOwnedBy() = false for different user")
	}
}

// ─── CanPublish ──────────────────────────────────────────────────────────────

func TestListing_CanPublish_Valid(t *testing.T) {
	l := testListing(StatusDraft, TypeLand)
	if !l.CanPublish() {
		t.Error("expected CanPublish() = true for valid listing")
	}
}

func TestListing_CanPublish_MissingTitle(t *testing.T) {
	l := testListing(StatusDraft, TypeLand)
	l.Title = ""
	if l.CanPublish() {
		t.Error("expected CanPublish() = false when title is empty")
	}
}

func TestListing_CanPublish_ZeroPrice(t *testing.T) {
	l := testListing(StatusDraft, TypeLand)
	l.PriceMXN = 0
	if l.CanPublish() {
		t.Error("expected CanPublish() = false when price is zero")
	}
}

func TestListing_CanPublish_MissingMunicipality(t *testing.T) {
	l := testListing(StatusDraft, TypeLand)
	l.Municipality = ""
	if l.CanPublish() {
		t.Error("expected CanPublish() = false when municipality is empty")
	}
}

func TestListing_CanPublish_MissingPropertyType(t *testing.T) {
	l := testListing(StatusDraft, TypeLand)
	l.PropertyType = ""
	if l.CanPublish() {
		t.Error("expected CanPublish() = false when property type is empty")
	}
}

// ─── PrimaryPhoto ────────────────────────────────────────────────────────────

func TestListing_PrimaryPhoto_NoMedia(t *testing.T) {
	l := testListing(StatusActive, TypeHouse)
	if l.PrimaryPhoto() != nil {
		t.Error("expected PrimaryPhoto() = nil when no media")
	}
}

func TestListing_PrimaryPhoto_FirstWhenNonePrimary(t *testing.T) {
	l := testListing(StatusActive, TypeHouse)
	l.Media = []ListingMedia{
		{ID: uuid.New(), URL: "https://example.com/1.jpg", IsPrimary: false, SortOrder: 0},
		{ID: uuid.New(), URL: "https://example.com/2.jpg", IsPrimary: false, SortOrder: 1},
	}

	photo := l.PrimaryPhoto()
	if photo == nil {
		t.Fatal("expected PrimaryPhoto() to return first photo, got nil")
	}
	if photo.URL != "https://example.com/1.jpg" {
		t.Errorf("expected first photo URL, got %q", photo.URL)
	}
}

func TestListing_PrimaryPhoto_ReturnsPrimary(t *testing.T) {
	l := testListing(StatusActive, TypeHouse)
	l.Media = []ListingMedia{
		{ID: uuid.New(), URL: "https://example.com/1.jpg", IsPrimary: false, SortOrder: 0},
		{ID: uuid.New(), URL: "https://example.com/primary.jpg", IsPrimary: true, SortOrder: 1},
	}

	photo := l.PrimaryPhoto()
	if photo == nil {
		t.Fatal("expected PrimaryPhoto() to return primary photo, got nil")
	}
	if photo.URL != "https://example.com/primary.jpg" {
		t.Errorf("expected primary photo URL, got %q", photo.URL)
	}
}

// ─── Property Types ──────────────────────────────────────────────────────────

func TestPropertyTypes_AllDefined(t *testing.T) {
	types := []PropertyType{
		TypeLand,
		TypeHouse,
		TypeRancho,
		TypeCommercial,
		TypeCabin,
	}

	for _, pt := range types {
		if pt == "" {
			t.Errorf("property type constant is empty string")
		}
	}
}

// ─── Listing Status ──────────────────────────────────────────────────────────

func TestListingStatus_AllDefined(t *testing.T) {
	statuses := []ListingStatus{
		StatusDraft,
		StatusActive,
		StatusUnderContract,
		StatusSold,
		StatusArchived,
	}

	for _, s := range statuses {
		if s == "" {
			t.Errorf("listing status constant is empty string")
		}
	}
}
