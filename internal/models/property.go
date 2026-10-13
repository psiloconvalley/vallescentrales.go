// internal/models/property.go
//
// PropertyImage is the shared media record used by listings for photo galleries.
// Historically a full "Property" struct also lived here (parallel to Listing),
// but Phase 1 consolidated the real estate model onto the simpler "Listing"
// in listing.go. PropertyImage remains because ListingMedia aliases it.
//
// Do not reintroduce the Property struct or the operation/legal-regime enum
// definitions here — they belong on Listing in listing.go.

package models

import (
	"time"

	"github.com/google/uuid"
)

type PropertyImage struct {
	ID         uuid.UUID `json:"id"          db:"id"`
	PropertyID uuid.UUID `json:"property_id" db:"property_id"`
	ListingID  uuid.UUID `json:"listing_id"  db:"listing_id"`
	URL        string    `json:"url"         db:"url"`
	Caption    string    `json:"caption"     db:"caption"`
	IsPrimary  bool      `json:"is_primary"  db:"is_primary"`
	SortOrder  int       `json:"sort_order"  db:"sort_order"`
	CreatedAt  time.Time `json:"created_at"  db:"created_at"`
}
