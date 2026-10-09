package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vallescentrales/internal/models"
	"vallescentrales/internal/slug"
)

type ListFilter struct {
	Municipality *string
	PropertyType *models.PropertyType
	MinPrice     *float64
	MaxPrice     *float64
	Status       *models.ListingStatus
	Featured     *bool
	OwnerID      *uuid.UUID
	Page         int
	PageSize     int
	SortBy       string
}

type CreateInput struct {
	OwnerID        uuid.UUID
	Title          string
	Slug           string
	Description    *string
	PropertyType   models.PropertyType
	PriceMXN       float64
	AreaM2         *float64
	ConstructionM2 *float64
	Municipality   string
	Community      *string
	Latitude       *float64
	Longitude      *float64
}

type ListingRepo struct {
	db *pgxpool.Pool
}

func NewListingRepo(db *pgxpool.Pool) *ListingRepo {
	return &ListingRepo{db: db}
}

func (r *ListingRepo) Create(ctx context.Context, l *models.Listing) error {
	query := `
		INSERT INTO listings (
			id, owner_id, title, slug, description, property_type, status,
			price_mxn, area_m2, construction_m2, municipality, community,
			latitude, longitude, is_featured, view_count, created_at, updated_at, published_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`
	_, err := r.db.Exec(ctx, query,
		l.ID, l.OwnerID, l.Title, l.Slug, l.Description, l.PropertyType, l.Status,
		l.PriceMXN, l.AreaM2, l.ConstructionM2, l.Municipality, l.Community,
		l.Latitude, l.Longitude, l.IsFeatured, l.ViewCount, l.CreatedAt, l.UpdatedAt, l.PublishedAt,
	)
	if err != nil {
		return fmt.Errorf("listing_repo: create: %w", err)
	}
	return nil
}

func (r *ListingRepo) CreateFromInput(ctx context.Context, input CreateInput) (*models.Listing, error) {
	now := time.Now().UTC()
	listingSlug := input.Slug
	if listingSlug == "" {
		listingSlug = slug.Generate(input.Title)
	}

	l := &models.Listing{
		ID:             uuid.New(),
		OwnerID:        input.OwnerID,
		Title:          input.Title,
		Slug:           listingSlug,
		Description:    input.Description,
		PropertyType:   input.PropertyType,
		Status:         models.StatusDraft,
		PriceMXN:       input.PriceMXN,
		AreaM2:         input.AreaM2,
		ConstructionM2: input.ConstructionM2,
		Municipality:   input.Municipality,
		Community:      input.Community,
		Latitude:       input.Latitude,
		Longitude:      input.Longitude,
		IsFeatured:     false,
		ViewCount:      0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := r.Create(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (r *ListingRepo) List(ctx context.Context, filter ListFilter) ([]*models.Listing, int, error) {
	var whereClauses []string
	var args []interface{}
	argIdx := 1

	if filter.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	} else {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, models.StatusActive)
		argIdx++
	}

	if filter.Municipality != nil && *filter.Municipality != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("municipality = $%d", argIdx))
		args = append(args, *filter.Municipality)
		argIdx++
	}

	if filter.PropertyType != nil && *filter.PropertyType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("property_type = $%d", argIdx))
		args = append(args, *filter.PropertyType)
		argIdx++
	}

	if filter.MinPrice != nil && *filter.MinPrice > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("price_mxn >= $%d", argIdx))
		args = append(args, *filter.MinPrice)
		argIdx++
	}

	if filter.MaxPrice != nil && *filter.MaxPrice > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("price_mxn <= $%d", argIdx))
		args = append(args, *filter.MaxPrice)
		argIdx++
	}

	if filter.OwnerID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("owner_id = $%d", argIdx))
		args = append(args, *filter.OwnerID)
		argIdx++
	}

	if filter.Featured != nil && *filter.Featured {
		whereClauses = append(whereClauses, fmt.Sprintf("is_featured = $%d", argIdx))
		args = append(args, true)
		argIdx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM listings %s", whereSQL)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("listing_repo: count list: %w", err)
	}

	orderSQL := "ORDER BY is_featured DESC, created_at DESC"
	switch filter.SortBy {
	case "price_asc":
		orderSQL = "ORDER BY price_mxn ASC, created_at DESC"
	case "price_desc":
		orderSQL = "ORDER BY price_mxn DESC, created_at DESC"
	case "recent":
		orderSQL = "ORDER BY created_at DESC"
	}

	limit := filter.PageSize
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	query := fmt.Sprintf(`
		SELECT id, owner_id, title, slug, description, property_type, status,
		       price_mxn, area_m2, construction_m2, municipality, community,
		       latitude, longitude, is_featured, view_count, created_at, updated_at, published_at
		FROM listings
		%s
		%s
		LIMIT $%d OFFSET $%d
	`, whereSQL, orderSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing_repo: list query: %w", err)
	}
	defer rows.Close()

	var listings []*models.Listing
	for rows.Next() {
		var l models.Listing
		err := rows.Scan(
			&l.ID, &l.OwnerID, &l.Title, &l.Slug, &l.Description, &l.PropertyType, &l.Status,
			&l.PriceMXN, &l.AreaM2, &l.ConstructionM2, &l.Municipality, &l.Community,
			&l.Latitude, &l.Longitude, &l.IsFeatured, &l.ViewCount, &l.CreatedAt, &l.UpdatedAt, &l.PublishedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("listing_repo: scan list: %w", err)
		}

		images, err := r.GetMedia(ctx, l.ID)
		if err == nil {
			l.Media = images
		}

		listings = append(listings, &l)
	}

	return listings, total, rows.Err()
}

func (r *ListingRepo) Update(ctx context.Context, l *models.Listing) error {
	l.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE listings SET
			title = $1, slug = $2, description = $3, property_type = $4,
			status = $5, price_mxn = $6, area_m2 = $7, construction_m2 = $8,
			municipality = $9, community = $10, latitude = $11, longitude = $12,
			is_featured = $13, updated_at = $14, published_at = $15
		WHERE id = $16
	`
	_, err := r.db.Exec(ctx, query,
		l.Title, l.Slug, l.Description, l.PropertyType, l.Status, l.PriceMXN,
		l.AreaM2, l.ConstructionM2, l.Municipality, l.Community, l.Latitude, l.Longitude,
		l.IsFeatured, l.UpdatedAt, l.PublishedAt, l.ID,
	)
	if err != nil {
		return fmt.Errorf("listing_repo: update: %w", err)
	}
	return nil
}

func (r *ListingRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Listing, error) {
	query := `
		SELECT id, owner_id, title, slug, description, property_type, status,
		       price_mxn, area_m2, construction_m2, municipality, community,
		       latitude, longitude, is_featured, view_count, created_at, updated_at, published_at
		FROM listings
		WHERE id = $1
	`
	var l models.Listing
	err := r.db.QueryRow(ctx, query, id).Scan(
		&l.ID, &l.OwnerID, &l.Title, &l.Slug, &l.Description, &l.PropertyType, &l.Status,
		&l.PriceMXN, &l.AreaM2, &l.ConstructionM2, &l.Municipality, &l.Community,
		&l.Latitude, &l.Longitude, &l.IsFeatured, &l.ViewCount, &l.CreatedAt, &l.UpdatedAt, &l.PublishedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("listing_repo: get by id: %w", err)
	}

	images, err := r.GetMedia(ctx, l.ID)
	if err == nil {
		l.Media = images
	}

	return &l, nil
}

func (r *ListingRepo) GetBySlug(ctx context.Context, slug string) (*models.Listing, error) {
	query := `
		SELECT id, owner_id, title, slug, description, property_type, status,
		       price_mxn, area_m2, construction_m2, municipality, community,
		       latitude, longitude, is_featured, view_count, created_at, updated_at, published_at
		FROM listings
		WHERE slug = $1
	`
	var l models.Listing
	err := r.db.QueryRow(ctx, query, slug).Scan(
		&l.ID, &l.OwnerID, &l.Title, &l.Slug, &l.Description, &l.PropertyType, &l.Status,
		&l.PriceMXN, &l.AreaM2, &l.ConstructionM2, &l.Municipality, &l.Community,
		&l.Latitude, &l.Longitude, &l.IsFeatured, &l.ViewCount, &l.CreatedAt, &l.UpdatedAt, &l.PublishedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("listing_repo: get by slug: %w", err)
	}

	images, err := r.GetMedia(ctx, l.ID)
	if err == nil {
		l.Media = images
	}

	return &l, nil
}

func (r *ListingRepo) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]*models.Listing, error) {
	query := `
		SELECT id, owner_id, title, slug, description, property_type, status,
		       price_mxn, area_m2, construction_m2, municipality, community,
		       latitude, longitude, is_featured, view_count, created_at, updated_at, published_at
		FROM listings
		WHERE owner_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("listing_repo: list by owner: %w", err)
	}
	defer rows.Close()

	var listings []*models.Listing
	for rows.Next() {
		var l models.Listing
		err := rows.Scan(
			&l.ID, &l.OwnerID, &l.Title, &l.Slug, &l.Description, &l.PropertyType, &l.Status,
			&l.PriceMXN, &l.AreaM2, &l.ConstructionM2, &l.Municipality, &l.Community,
			&l.Latitude, &l.Longitude, &l.IsFeatured, &l.ViewCount, &l.CreatedAt, &l.UpdatedAt, &l.PublishedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("listing_repo: scan list by owner: %w", err)
		}

		images, err := r.GetMedia(ctx, l.ID)
		if err == nil {
			l.Media = images
		}

		listings = append(listings, &l)
	}

	return listings, rows.Err()
}

func (r *ListingRepo) AddMedia(ctx context.Context, listingID uuid.UUID, mediaURL string, caption string, isPrimary bool, sortOrder int) (*models.ListingMedia, error) {
	id := uuid.New()
	now := time.Now().UTC()

	query := `
		INSERT INTO listing_media (id, listing_id, url, caption, is_primary, sort_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query, id, listingID, mediaURL, caption, isPrimary, sortOrder, now)
	if err != nil {
		return nil, fmt.Errorf("listing_repo: add media: %w", err)
	}

	return &models.ListingMedia{
		ID:        id,
		ListingID: listingID,
		URL:       mediaURL,
		Caption:   caption,
		IsPrimary: isPrimary,
		SortOrder: sortOrder,
		CreatedAt: now,
	}, nil
}

func (r *ListingRepo) GetMedia(ctx context.Context, listingID uuid.UUID) ([]models.ListingMedia, error) {
	query := `
		SELECT id, listing_id, url, COALESCE(caption, ''), is_primary, sort_order, created_at
		FROM listing_media
		WHERE listing_id = $1
		ORDER BY sort_order ASC, created_at ASC
	`
	rows, err := r.db.Query(ctx, query, listingID)
	if err != nil {
		return nil, fmt.Errorf("listing_repo: get media: %w", err)
	}
	defer rows.Close()

	var media []models.ListingMedia
	for rows.Next() {
		var m models.ListingMedia
		if err := rows.Scan(&m.ID, &m.ListingID, &m.URL, &m.Caption, &m.IsPrimary, &m.SortOrder, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("listing_repo: scan media: %w", err)
		}
		media = append(media, m)
	}

	return media, rows.Err()
}

func (r *ListingRepo) ClearMedia(ctx context.Context, listingID uuid.UUID) error {
	query := `DELETE FROM listing_media WHERE listing_id = $1`
	_, err := r.db.Exec(ctx, query, listingID)
	if err != nil {
		return fmt.Errorf("listing_repo: clear media: %w", err)
	}
	return nil
}

func (r *ListingRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM listings WHERE slug = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("listing_repo: slug exists: %w", err)
	}
	return exists, nil
}

func (r *ListingRepo) Publish(ctx context.Context, id uuid.UUID) (*models.Listing, error) {
	now := time.Now().UTC()
	query := `
		UPDATE listings
		SET status = $1, published_at = $2, updated_at = $3
		WHERE id = $4
	`
	res, err := r.db.Exec(ctx, query, models.StatusActive, now, now, id)
	if err != nil {
		return nil, fmt.Errorf("listing_repo: publish: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *ListingRepo) Archive(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	query := `
		UPDATE listings
		SET status = $1, updated_at = $2
		WHERE id = $3
	`
	_, err := r.db.Exec(ctx, query, models.StatusArchived, now, id)
	if err != nil {
		return fmt.Errorf("listing_repo: archive: %w", err)
	}
	return nil
}

func (r *ListingRepo) IncrementViewCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE listings SET view_count = view_count + 1 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("listing_repo: increment view count: %w", err)
	}
	return nil
}

func (r *ListingRepo) CreateProperty(ctx context.Context, p *models.Property, photos []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("listing_repo: begin tx: %w", err)
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
		return fmt.Errorf("listing_repo: insert property: %w", err)
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
			return fmt.Errorf("listing_repo: insert property image: %w", err)
		}
	}

	return tx.Commit(ctx)
}
