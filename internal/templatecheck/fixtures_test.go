package templatecheck

import (
	"time"

	"vallescentrales/internal/models"

	"github.com/google/uuid"
)

// ─────────────────────────────────────────────────────────────────
// Pointer helpers — Go has no literal syntax for &"string", so we
// use tiny helpers to construct pointer fields cleanly in fixtures.
// ─────────────────────────────────────────────────────────────────

func stringPtr(s string) *string     { return &s }
func float64Ptr(f float64) *float64  { return &f }
func timePtr(t time.Time) *time.Time { return &t }

// ─────────────────────────────────────────────────────────────────
// fakeOwner builds a realistic listing owner (the seller).
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
// fakeViewer builds a logged-in visitor browsing listings.
// ─────────────────────────────────────────────────────────────────

func fakeViewer() *models.User {
	return &models.User{
		ID:                  uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Email:               "comprador@example.com",
		FullName:            "Carlos Hernández",
		DisplayName:         stringPtr("Carlos H."),
		AvatarURL:           stringPtr("https://example.com/carlos.jpg"),
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
// fakePhoto returns a populated PropertyImage for listings that need media.
// ─────────────────────────────────────────────────────────────────

func fakePhoto(listingID uuid.UUID, isPrimary bool, sortOrder int) models.PropertyImage {
	return models.PropertyImage{
		ID:         uuid.New(),
		PropertyID: listingID,
		ListingID:  listingID,
		URL:        "https://example.com/photo.jpg",
		Caption:    "Vista frontal de la propiedad",
		IsPrimary:  isPrimary,
		SortOrder:  sortOrder,
		CreatedAt:  time.Now().Add(-7 * 24 * time.Hour),
	}
}

// ─────────────────────────────────────────────────────────────────
// fakeListing builds a realistic property listing. Optionally includes
// a primary photo so templates that render card images work correctly.
// ─────────────────────────────────────────────────────────────────

func fakeListing(ownerID uuid.UUID, withPhoto bool) *models.Listing {
	id := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	publishedAt := time.Now().Add(-7 * 24 * time.Hour)

	listing := &models.Listing{
		ID:             id,
		OwnerID:        ownerID,
		Title:          "Casa de campo con terreno en San Felipe del Agua",
		Slug:           "casa-de-campo-san-felipe-del-agua",
		Description:    stringPtr("Hermosa casa de campo con amplio terreno, árboles frutales y vista a la sierra."),
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
		Media:          []models.ListingMedia{},
	}

	if withPhoto {
		listing.Media = []models.ListingMedia{
			fakePhoto(id, true, 0),
		}
	}

	return listing
}

// ─────────────────────────────────────────────────────────────────
// fakeListingDetailData builds the full data payload for listing_detail.tmpl.
// ─────────────────────────────────────────────────────────────────

func fakeListingDetailData() map[string]any {
	owner := fakeOwner()
	viewer := fakeViewer()
	listing := fakeListing(owner.ID, true)

	return map[string]any{
		"Listing": listing,
		"Owner":   owner,
		"User":    viewer,
		"URL":     "https://bienesraices.vallescentrales.com/propiedades/" + listing.Slug,
	}
}

// ─────────────────────────────────────────────────────────────────
// fakeListingCardData returns a *models.Listing directly, since the
// listing_card partial is called as {{template "listing_card" .Listing}}
// and receives a Listing as its root context.
// ─────────────────────────────────────────────────────────────────

func fakeListingCardData() *models.Listing {
	owner := fakeOwner()
	return fakeListing(owner.ID, true)
}

// ─────────────────────────────────────────────────────────────────
// fakeDashboardData builds the payload for dashboard.tmpl which iterates
// over .Listings and reads .User fields.
// ─────────────────────────────────────────────────────────────────

func fakeDashboardData() map[string]any {
	user := fakeOwner()

	listings := []*models.Listing{
		fakeListing(user.ID, true),
		fakeListing(user.ID, false),
		fakeListing(user.ID, true),
	}

	return map[string]any{
		"User":     user,
		"Listings": listings,
	}
}

// ─────────────────────────────────────────────────────────────────
// fakeWizardPropertyStep1Data builds the payload for the first wizard
// step. The wizard engine passes prior-step form values via StepData.
// ─────────────────────────────────────────────────────────────────

func fakeWizardPropertyStep1Data() map[string]any {
	return map[string]any{
		"StepData": map[string]any{
			"operation_type": "sale",
			"property_type":  "house",
			"legal_regime":   "escritura_publica",
		},
	}
}
