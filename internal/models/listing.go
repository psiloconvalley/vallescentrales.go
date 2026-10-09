package models

import (
	"time"

	"github.com/google/uuid"
)

type ListingStatus string

const (
	StatusDraft         ListingStatus = "draft"
	StatusActive        ListingStatus = "active"
	StatusUnderContract ListingStatus = "under_contract"
	StatusSold          ListingStatus = "sold"
	StatusArchived      ListingStatus = "archived"
)

type PropertyType string

const (
	TypeLand       PropertyType = "land"
	TypeHouse      PropertyType = "house"
	TypeRancho     PropertyType = "rancho"
	TypeCommercial PropertyType = "commercial"
	TypeCabin      PropertyType = "cabin"
)

type Listing struct {
	ID             uuid.UUID     `db:"id"              json:"id"`
	OwnerID        uuid.UUID     `db:"owner_id"        json:"owner_id"`
	Title          string        `db:"title"           json:"title"`
	Slug           string        `db:"slug"            json:"slug"`
	Description    *string       `db:"description"     json:"description,omitempty"`
	PropertyType   PropertyType  `db:"property_type"   json:"property_type"`
	Status         ListingStatus `db:"status"          json:"status"`
	PriceMXN       float64       `db:"price_mxn"       json:"price_mxn"`
	AreaM2         *float64      `db:"area_m2"         json:"area_m2,omitempty"`
	ConstructionM2 *float64      `db:"construction_m2" json:"construction_m2,omitempty"`
	Municipality   string        `db:"municipality"    json:"municipality"`
	Community      *string       `db:"community"       json:"community,omitempty"`
	Latitude       *float64      `db:"latitude"        json:"latitude,omitempty"`
	Longitude      *float64      `db:"longitude"       json:"longitude,omitempty"`
	IsFeatured     bool          `db:"is_featured"     json:"is_featured"`
	ViewCount      int           `db:"view_count"      json:"view_count"`
	CreatedAt      time.Time     `db:"created_at"      json:"created_at"`
	UpdatedAt      time.Time     `db:"updated_at"      json:"updated_at"`
	PublishedAt    *time.Time    `db:"published_at"    json:"published_at,omitempty"`

	Media []ListingMedia `db:"-" json:"media,omitempty"`
}

type ListingMedia = PropertyImage

func (l *Listing) IsPublic() bool {
	return l.Status == StatusActive
}

func (l *Listing) IsOwnedBy(userID uuid.UUID) bool {
	return l.OwnerID == userID
}

func (l *Listing) PrimaryPhoto() *ListingMedia {
	for i := range l.Media {
		if l.Media[i].IsPrimary {
			return &l.Media[i]
		}
	}
	if len(l.Media) > 0 {
		return &l.Media[0]
	}
	return nil
}

func (l *Listing) CanPublish() bool {
	return l.Title != "" &&
		l.PropertyType != "" &&
		l.PriceMXN > 0 &&
		l.Municipality != ""
}
