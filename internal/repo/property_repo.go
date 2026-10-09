package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"vallescentrales/internal/models"
)

type PropertyRepo struct {
	db *pgxpool.Pool
}

func NewPropertyRepo(db *pgxpool.Pool) *PropertyRepo {
	return &PropertyRepo{db: db}
}

func (r *PropertyRepo) Create(ctx context.Context, p *models.Property, photos []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("property_repo: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	queryProp := `
		INSERT INTO properties (
			id, owner_id, title, slug, description, operation_type, property_type,
			legal_regime, price_mxn, currency, area_total_m2, area_built_m2,
			bedrooms, bathrooms, parking_spaces, municipality, community, address,
			latitude, longitude, status, listing_tier, view_count,
			contact_phone, contact_whatsapp, show_phone, show_whatsapp,
			created_at, updated_at, published_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18,
			$19, $20, $21, $22, $23,
			$24, $25, $26, $27,
			$28, $29, $30
		)
	`
	_, err = tx.Exec(ctx, queryProp,
		p.ID, p.OwnerID, p.Title, p.Slug, p.Description, p.OperationType, p.PropertyType,
		p.LegalRegime, p.PriceMXN, p.Currency, p.AreaTotalM2, p.AreaBuiltM2,
		p.Bedrooms, p.Bathrooms, p.ParkingSpaces, p.Municipality, p.Community, p.Address,
		p.Latitude, p.Longitude, p.Status, p.ListingTier, p.ViewCount,
		p.ContactPhone, p.ContactWhatsApp, p.ShowPhone, p.ShowWhatsApp,
		p.CreatedAt, p.UpdatedAt, p.PublishedAt,
	)
	if err != nil {
		return fmt.Errorf("property_repo: insert property: %w", err)
	}

	queryPhoto := `
		INSERT INTO property_images (id, property_id, listing_id, url, is_primary, sort_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	now := time.Now().UTC()
	for idx, photoURL := range photos {
		isPrimary := (idx == 0)
		_, err = tx.Exec(ctx, queryPhoto,
			uuid.New(),
			p.ID,
			p.ID,
			photoURL,
			isPrimary,
			idx,
			now,
		)
		if err != nil {
			return fmt.Errorf("property_repo: insert property image: %w", err)
		}
	}

	return tx.Commit(ctx)
}
