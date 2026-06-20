// internal/repo/listing_repo.go
// All SQL for the listings and listing_media tables.
// Rule 42: repo = SQL only. No business logic. No HTTP.
// Rule 8:  Always parameterized queries. Never string interpolation.
// Rule 9:  Column names verified against live schema 2026-06-20.

package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vallescentrales/internal/models"
)

// ListingRepo handles all database operations for listings and listing_media.
type ListingRepo struct {
	db *pgxpool.Pool
}

// NewListingRepo creates a new ListingRepo.
func NewListingRepo(db *pgxpool.Pool) *ListingRepo {
	return &ListingRepo{db: db}
}

// CreateInput holds the fields required to insert a new listing.
type CreateInput struct {
	OwnerID      uuid.UUID
	Title        string
	Slug         string
	PropertyType models.PropertyType
	PriceMXN     float64
	Municipality string
}

// Create inserts a new listing in draft status.
// Slug must be generated and verified unique before calling this.
func (r *ListingRepo) Create(ctx context.Context, input CreateInput) (*models.Listing, error) {
	query := `
		INSERT INTO listings (owner_id, title, slug, property_type, price_mxn, municipality)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, owner_id, title, slug, description, property_type, status,
		          price_mxn, area_m2, construction_m2, municipality, community,
		          latitude, longitude, is_featured, view_count,
		          created_at, updated_at, published_at`

	listing := &models.Listing{}
	err := r.db.QueryRow(ctx, query,
		input.OwnerID,
		input.Title,
		input.Slug,
		input.PropertyType,
		input.PriceMXN,
		input.Municipality,
	).Scan(
		&listing.ID,
		&listing.OwnerID,
		&listing.Title,
		&listing.Slug,
		&listing.Description,
		&listing.PropertyType,
		&listing.Status,
		&listing.PriceMXN,
		&listing.AreaM2,
		&listing.ConstructionM2,
		&listing.Municipality,
		&listing.Community,
		&listing.Latitude,
		&listing.Longitude,
		&listing.IsFeatured,
		&listing.ViewCount,
		&listing.CreatedAt,
		&listing.UpdatedAt,
		&listing.PublishedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("listing_repo.Create: %w", err)
	}

	return listing, nil
}

// GetByID fetches a single listing by primary key.
func (r *ListingRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Listing, error) {
	query := `
		SELECT id, owner_id, title, slug, description, property_type, status,
		       price_mxn, area_m2, construction_m2, municipality, community,
		       latitude, longitude, is_featured, view_count,
		       created_at, updated_at, published_at
		FROM listings
		WHERE id = $1`

	listing := &models.Listing{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&listing.ID,
		&listing.OwnerID,
		&listing.Title,
		&listing.Slug,
		&listing.Description,
		&listing.PropertyType,
		&listing.Status,
		&listing.PriceMXN,
		&listing.AreaM2,
		&listing.ConstructionM2,
		&listing.Municipality,
		&listing.Community,
		&listing.Latitude,
		&listing.Longitude,
		&listing.IsFeatured,
		&listing.ViewCount,
		&listing.CreatedAt,
		&listing.UpdatedAt,
		&listing.PublishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("listing_repo.GetByID: %w", err)
	}

	return listing, nil
}

// GetBySlug fetches a single listing by its URL slug.
func (r *ListingRepo) GetBySlug(ctx context.Context, slug string) (*models.Listing, error) {
	query := `
		SELECT id, owner_id, title, slug, description, property_type, status,
		       price_mxn, area_m2, construction_m2, municipality, community,
		       latitude, longitude, is_featured, view_count,
		       created_at, updated_at, published_at
		FROM listings
		WHERE slug = $1`

	listing := &models.Listing{}
	err := r.db.QueryRow(ctx, query, slug).Scan(
		&listing.ID,
		&listing.OwnerID,
		&listing.Title,
		&listing.Slug,
		&listing.Description,
		&listing.PropertyType,
		&listing.Status,
		&listing.PriceMXN,
		&listing.AreaM2,
		&listing.ConstructionM2,
		&listing.Municipality,
		&listing.Community,
		&listing.Latitude,
		&listing.Longitude,
		&listing.IsFeatured,
		&listing.ViewCount,
		&listing.CreatedAt,
		&listing.UpdatedAt,
		&listing.PublishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("listing_repo.GetBySlug: %w", err)
	}

	return listing, nil
}

// ListFilter defines the available filters for the public listing browse page.
type ListFilter struct {
	Municipality *string
	PropertyType *models.PropertyType
	MinPrice     *float64
	MaxPrice     *float64
	Page         int
	PageSize     int
}

// List returns active listings with optional filters.
// Only returns listings with status = 'active' — never draft or archived.
func (r *ListingRepo) List(ctx context.Context, f ListFilter) ([]*models.Listing, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 50 {
		f.PageSize = 20
	}
	offset := (f.Page - 1) * f.PageSize

	// Count query
	countQuery := `
		SELECT COUNT(*)
		FROM listings
		WHERE status = 'active'
		  AND ($1::text IS NULL OR municipality = $1)
		  AND ($2::text IS NULL OR property_type = $2::property_type)
		  AND ($3::numeric IS NULL OR price_mxn >= $3)
		  AND ($4::numeric IS NULL OR price_mxn <= $4)`

	var total int
	err := r.db.QueryRow(ctx, countQuery,
		f.Municipality,
		f.PropertyType,
		f.MinPrice,
		f.MaxPrice,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("listing_repo.List count: %w", err)
	}

	if total == 0 {
		return []*models.Listing{}, 0, nil
	}

	// Data query
	dataQuery := `
		SELECT id, owner_id, title, slug, description, property_type, status,
		       price_mxn, area_m2, construction_m2, municipality, community,
		       latitude, longitude, is_featured, view_count,
		       created_at, updated_at, published_at
		FROM listings
		WHERE status = 'active'
		  AND ($1::text IS NULL OR municipality = $1)
		  AND ($2::text IS NULL OR property_type = $2::property_type)
		  AND ($3::numeric IS NULL OR price_mxn >= $3)
		  AND ($4::numeric IS NULL OR price_mxn <= $4)
		ORDER BY is_featured DESC, published_at DESC
		LIMIT $5 OFFSET $6`

	rows, err := r.db.Query(ctx, dataQuery,
		f.Municipality,
		f.PropertyType,
		f.MinPrice,
		f.MaxPrice,
		f.PageSize,
		offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("listing_repo.List query: %w", err)
	}
	defer rows.Close()

	listings := make([]*models.Listing, 0)
	for rows.Next() {
		listing := &models.Listing{}
		err := rows.Scan(
			&listing.ID,
			&listing.OwnerID,
			&listing.Title,
			&listing.Slug,
			&listing.Description,
			&listing.PropertyType,
			&listing.Status,
			&listing.PriceMXN,
			&listing.AreaM2,
			&listing.ConstructionM2,
			&listing.Municipality,
			&listing.Community,
			&listing.Latitude,
			&listing.Longitude,
			&listing.IsFeatured,
			&listing.ViewCount,
			&listing.CreatedAt,
			&listing.UpdatedAt,
			&listing.PublishedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("listing_repo.List scan: %w", err)
		}
		listings = append(listings, listing)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("listing_repo.List rows: %w", err)
	}

	return listings, total, nil
}

// ListByOwner returns all listings for a specific owner (dashboard view).
// Returns all statuses — owner sees their drafts and archived listings too.
func (r *ListingRepo) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]*models.Listing, error) {
	query := `
		SELECT id, owner_id, title, slug, description, property_type, status,
		       price_mxn, area_m2, construction_m2, municipality, community,
		       latitude, longitude, is_featured, view_count,
		       created_at, updated_at, published_at
		FROM listings
		WHERE owner_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("listing_repo.ListByOwner: %w", err)
	}
	defer rows.Close()

	listings := make([]*models.Listing, 0)
	for rows.Next() {
		listing := &models.Listing{}
		err := rows.Scan(
			&listing.ID,
			&listing.OwnerID,
			&listing.Title,
			&listing.Slug,
			&listing.Description,
			&listing.PropertyType,
			&listing.Status,
			&listing.PriceMXN,
			&listing.AreaM2,
			&listing.ConstructionM2,
			&listing.Municipality,
			&listing.Community,
			&listing.Latitude,
			&listing.Longitude,
			&listing.IsFeatured,
			&listing.ViewCount,
			&listing.CreatedAt,
			&listing.UpdatedAt,
			&listing.PublishedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("listing_repo.ListByOwner scan: %w", err)
		}
		listings = append(listings, listing)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing_repo.ListByOwner rows: %w", err)
	}

	return listings, nil
}

// Publish sets status to active and records published_at timestamp.
// ADR-004: draft → active transition.
func (r *ListingRepo) Publish(ctx context.Context, id uuid.UUID) (*models.Listing, error) {
	query := `
		UPDATE listings
		SET status       = 'active',
		    published_at = NOW()
		WHERE id = $1
		  AND status = 'draft'
		RETURNING id, owner_id, title, slug, description, property_type, status,
		          price_mxn, area_m2, construction_m2, municipality, community,
		          latitude, longitude, is_featured, view_count,
		          created_at, updated_at, published_at`

	listing := &models.Listing{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&listing.ID,
		&listing.OwnerID,
		&listing.Title,
		&listing.Slug,
		&listing.Description,
		&listing.PropertyType,
		&listing.Status,
		&listing.PriceMXN,
		&listing.AreaM2,
		&listing.ConstructionM2,
		&listing.Municipality,
		&listing.Community,
		&listing.Latitude,
		&listing.Longitude,
		&listing.IsFeatured,
		&listing.ViewCount,
		&listing.CreatedAt,
		&listing.UpdatedAt,
		&listing.PublishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("listing_repo.Publish: %w", err)
	}

	return listing, nil
}

// Archive sets status to archived — soft delete.
func (r *ListingRepo) Archive(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE listings
		SET status = 'archived'
		WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("listing_repo.Archive: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// IncrementViewCount adds one to the view counter.
// Fire and forget — called on every public listing page load.
func (r *ListingRepo) IncrementViewCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE listings SET view_count = view_count + 1 WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("listing_repo.IncrementViewCount: %w", err)
	}

	return nil
}

// SlugExists returns true if a slug is already taken.
// Used by the service layer before inserting a new listing.
func (r *ListingRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM listings WHERE slug = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("listing_repo.SlugExists: %w", err)
	}

	return exists, nil
}

// AddMedia inserts a media record for a listing.
func (r *ListingRepo) AddMedia(ctx context.Context, listingID uuid.UUID, url, storageKey string, isPrimary bool, sortOrder int) (*models.ListingMedia, error) {
	query := `
		INSERT INTO listing_media (listing_id, url, storage_key, is_primary, sort_order)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, listing_id, url, storage_key, is_primary, sort_order, alt_text, created_at`

	media := &models.ListingMedia{}
	err := r.db.QueryRow(ctx, query,
		listingID,
		url,
		storageKey,
		isPrimary,
		sortOrder,
	).Scan(
		&media.ID,
		&media.ListingID,
		&media.URL,
		&media.StorageKey,
		&media.IsPrimary,
		&media.SortOrder,
		&media.AltText,
		&media.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("listing_repo.AddMedia: %w", err)
	}

	return media, nil
}

// GetMedia returns all media for a listing ordered by sort_order.
func (r *ListingRepo) GetMedia(ctx context.Context, listingID uuid.UUID) ([]models.ListingMedia, error) {
	query := `
		SELECT id, listing_id, url, storage_key, is_primary, sort_order, alt_text, created_at
		FROM listing_media
		WHERE listing_id = $1
		ORDER BY sort_order ASC, created_at ASC`

	rows, err := r.db.Query(ctx, query, listingID)
	if err != nil {
		return nil, fmt.Errorf("listing_repo.GetMedia: %w", err)
	}
	defer rows.Close()

	media := make([]models.ListingMedia, 0)
	for rows.Next() {
		m := models.ListingMedia{}
		err := rows.Scan(
			&m.ID,
			&m.ListingID,
			&m.URL,
			&m.StorageKey,
			&m.IsPrimary,
			&m.SortOrder,
			&m.AltText,
			&m.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("listing_repo.GetMedia scan: %w", err)
		}
		media = append(media, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listing_repo.GetMedia rows: %w", err)
	}

	return media, nil
}
