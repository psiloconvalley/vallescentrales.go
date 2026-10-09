package models

import (
	"time"

	"github.com/google/uuid"
)

type OperationType string

const (
	OperationSale OperationType = "sale"
	OperationRent OperationType = "rent"
)

type LegalRegime string

const (
	RegimeEscritura LegalRegime = "escritura_publica"
	RegimeEjidal    LegalRegime = "ejidal"
	RegimeComunal   LegalRegime = "comunal"
	RegimePosesion  LegalRegime = "posesion"
	RegimeOtro      LegalRegime = "otro"
)

type Property struct {
	ID              uuid.UUID     `json:"id" db:"id"`
	OwnerID         uuid.UUID     `json:"owner_id" db:"owner_id"`
	Title           string        `json:"title" db:"title"`
	Slug            string        `json:"slug" db:"slug"`
	Description     *string       `json:"description,omitempty" db:"description"`
	OperationType   OperationType `json:"operation_type" db:"operation_type"`
	PropertyType    PropertyType  `json:"property_type" db:"property_type"`
	LegalRegime     LegalRegime   `json:"legal_regime" db:"legal_regime"`
	PriceMXN        float64       `json:"price_mxn" db:"price_mxn"`
	Currency        string        `json:"currency" db:"currency"`
	AreaTotalM2     *float64      `json:"area_total_m2,omitempty" db:"area_total_m2"`
	AreaBuiltM2     *float64      `json:"area_built_m2,omitempty" db:"area_built_m2"`
	Bedrooms        *int          `json:"bedrooms,omitempty" db:"bedrooms"`
	Bathrooms       *float64      `json:"bathrooms,omitempty" db:"bathrooms"`
	ParkingSpaces   *int          `json:"parking_spaces,omitempty" db:"parking_spaces"`
	Municipality    string        `json:"municipality" db:"municipality"`
	Community       *string       `json:"community,omitempty" db:"community"`
	Address         *string       `json:"address,omitempty" db:"address"`
	Latitude        *float64      `json:"latitude,omitempty" db:"latitude"`
	Longitude       *float64      `json:"longitude,omitempty" db:"longitude"`
	Status          ListingStatus `json:"status" db:"status"`
	ListingTier     string        `json:"listing_tier" db:"listing_tier"`
	ViewCount       int           `json:"view_count" db:"view_count"`
	ContactPhone    *string       `json:"contact_phone,omitempty" db:"contact_phone"`
	ContactWhatsApp *string       `json:"contact_whatsapp,omitempty" db:"contact_whatsapp"`
	ShowPhone       bool          `json:"show_phone" db:"show_phone"`
	ShowWhatsApp    bool          `json:"show_whatsapp" db:"show_whatsapp"`
	CreatedAt       time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at" db:"updated_at"`
	PublishedAt     *time.Time    `json:"published_at,omitempty" db:"published_at"`

	Images []PropertyImage `json:"images,omitempty" db:"-"`
}

type PropertyImage struct {
	ID         uuid.UUID `json:"id" db:"id"`
	PropertyID uuid.UUID `json:"property_id" db:"property_id"`
	ListingID  uuid.UUID `json:"listing_id" db:"listing_id"`
	URL        string    `json:"url" db:"url"`
	Caption    string    `json:"caption" db:"caption"`
	IsPrimary  bool      `json:"is_primary" db:"is_primary"`
	SortOrder  int       `json:"sort_order" db:"sort_order"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

func (p *Property) IsPublic() bool {
	return p.Status == StatusActive
}

func (p *Property) IsOwnedBy(userID uuid.UUID) bool {
	return p.OwnerID == userID
}
