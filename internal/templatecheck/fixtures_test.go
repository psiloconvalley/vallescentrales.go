package templatecheck

import (
	"time"

	"vallescentrales/internal/models"

	"github.com/google/uuid"
)

// ─────────────────────────────────────────────────────────────────
// Pointer helpers — Go has no literal syntax for &"string", so we
// use tiny generic-feeling helpers to construct pointer fields
// cleanly inside fixtures.
// ─────────────────────────────────────────────────────────────────

func stringPtr(s string) *string          { return &s }
func float64Ptr(f float64) *float64       { return &f }
func timePtr(t time.Time) *time.Time      { return &t }

// ─────────────────────────────────────────────────────────────────
// fakeOwner builds a realistic listing owner (the seller).
// Returns *models.User because DisplayNameOrFull has a pointer receiver.
// ─────────────────────────────────────────────────────────────────

func fakeOwner() *models.User {
	return &models.User{
		ID:                  uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Email:               "maria.vendedora@example.com",
		FullName:            "María García López",
		DisplayName:         stringPtr("María G."),
		Phone:               stringPtr("+52 951 123 4567"),
		WhatsApp:            stringPtr("+52 951 123 4567"),
		ShowPhone:           true,
		ShowWhatsApp:        true,
		Role:                models.RoleUser,
		UserType:            "owner",
		IsVerified:          true,
		AuthProvider:        models.AuthProviderGoogle,
		AvatarURL:           stringPtr("https://example.com/avatar.jpg"),
		Bio:                 stringPtr("Propietaria en Oaxaca de Juárez."),
		Municipality:        stringPtr("Oaxaca de Juárez"),
		AgencyName:          stringPtr("Casa Oaxaca Inmuebles"),
		Languages:           []string{"es", "en"},
		NotifyEmail:         true,
		PreferredLang:       "es",
		OnboardingCompleted: true,
		CreatedAt:           time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt:           time.Now(),
	}
}

// ─────────────────────────────────────────────────────────────────
// fakeViewer builds a logged-in visitor (the buyer looking at a listing).
// ─────────────────────────────────────────────────────────────────

func fakeViewer() *models.User {
	return &models.User{
		ID:                  uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Email:               "comprador@example.com",
		FullName:            "Carlos Hernández",
		Role:                models.RoleUser,
		UserType:            "seeker",
		AuthProvider:        models.AuthProviderGoogle,
		PreferredLang:       "es",
		OnboardingCompleted: true,
		CreatedAt:           time.Now().Add(-5 * 24 * time.Hour),
		UpdatedAt:           time.Now(),
	}
}

// ─────────────────────────────────────────────────────────────────
// fakeListing builds a realistic property listing with every optional
// pointer field populated, so render tests exercise the full code path.
// ─────────────────────────────────────────────────────────────────

func fakeListing(ownerID uuid.UUID) *models.Listing {
	publishedAt := time.Now().Add(-7 * 24 * time.Hour)

	return &models.Listing{
		ID:             uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		OwnerID:        ownerID,
		Title:          "Casa de campo con terreno en San Felipe del Agua",
		Slug:           "casa-de-campo-san-felipe-del-agua",
		Description:    stringPtr("Hermosa casa de campo con amplio terreno, árboles frutales y vista a la sierra. Zona tranquila y segura, ideal para familias que buscan calidad de vida en Oaxaca."),
		PropertyType:   models.TypeHouse,
		OperationType:  models.OperationSale,
		LegalRegime:    models.LegalEscrituraPublica,
		Status:         models.StatusActive,
		PriceMXN:       3500000,
		AreaM2:         float64Ptr(850),
		ConstructionM2: float64Ptr(220),
		Municipality:   "San Felipe del Agua",
		Community:      stringPtr("Colonia Reforma"),
		Latitude:       float64Ptr(17.0732),
		Longitude:      float64Ptr(-96.7266),
		IsFeatured:     true,
		ViewCount:      42,
		CreatedAt:      time.Now().Add(-14 * 24 * time.Hour),
		UpdatedAt:      time.Now().Add(-1 * 24 * time.Hour),
		PublishedAt:    timePtr(publishedAt),
		Media:          []models.ListingMedia{}, // empty for now; template should handle gracefully
	}
}

// ─────────────────────────────────────────────────────────────────
// fakeListingDetailData builds the full data payload that
// listing_detail.tmpl expects: Listing, Owner, User (viewer), URL.
// ─────────────────────────────────────────────────────────────────

func fakeListingDetailData() map[string]any {
	owner := fakeOwner()
	viewer := fakeViewer()
	listing := fakeListing(owner.ID)

	return map[string]any{
		"Listing": listing,
		"Owner":   owner,
		"User":    viewer,
		"URL":     "https://bienesraices.vallescentrales.com/propiedades/" + listing.Slug,
	}
}
