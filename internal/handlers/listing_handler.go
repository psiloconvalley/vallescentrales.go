// internal/handlers/listing_handler.go
// Public and protected listing endpoints.
// Rule 42: handlers = HTTP only. No SQL. No business logic.

package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"vallescentrales/internal/middleware"
	"vallescentrales/internal/models"
	"vallescentrales/internal/repo"
)

// ListingHandler handles all listing HTTP endpoints.
type ListingHandler struct {
	listings *repo.ListingRepo
}

// NewListingHandler creates a ListingHandler.
func NewListingHandler(listings *repo.ListingRepo) *ListingHandler {
	return &ListingHandler{listings: listings}
}

// HandleHome serves the homepage.
// Returns JSON stub — replaced with template in Session 010.
func (h *ListingHandler) HandleHome(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{"page": "home"})
}

// HandleListListings returns active listings with optional filters.
// Query params: municipality, property_type, min_price, max_price, page, page_size
func (h *ListingHandler) HandleListListings(w http.ResponseWriter, r *http.Request) {
	filter := repo.ListFilter{
		Page:     1,
		PageSize: 20,
	}

	if v := r.URL.Query().Get("municipality"); v != "" {
		filter.Municipality = &v
	}

	if v := r.URL.Query().Get("property_type"); v != "" {
		pt := models.PropertyType(v)
		filter.PropertyType = &pt
	}

	if v := r.URL.Query().Get("min_price"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			filter.MinPrice = &f
		}
	}

	if v := r.URL.Query().Get("max_price"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			filter.MaxPrice = &f
		}
	}

	if v := r.URL.Query().Get("page"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			filter.Page = i
		}
	}

	listings, total, err := h.listings.List(r.Context(), filter)
	if err != nil {
		slog.Error("failed to list listings", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load listings")
		return
	}

	respond(w, http.StatusOK, map[string]any{
		"listings":  listings,
		"total":     total,
		"page":      filter.Page,
		"page_size": filter.PageSize,
	})
}

// HandleGetListing returns a single listing by slug.
// Increments view count on every public load.
func (h *ListingHandler) HandleGetListing(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		respondError(w, http.StatusBadRequest, "missing listing slug")
		return
	}

	listing, err := h.listings.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			respondError(w, http.StatusNotFound, "listing not found")
			return
		}
		slog.Error("failed to get listing", "slug", slug, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load listing")
		return
	}

	// Only public listings visible to unauthenticated users
	user := middleware.UserFromContext(r.Context())
	if !listing.IsPublic() && (user == nil || !listing.IsOwnedBy(user.ID)) {
		respondError(w, http.StatusNotFound, "listing not found")
		return
	}

	// Load media
	media, err := h.listings.GetMedia(r.Context(), listing.ID)
	if err != nil {
		slog.Warn("failed to load listing media", "listing_id", listing.ID, "error", err)
	} else {
		listing.Media = media
	}

	// Increment view count — fire and forget, non-fatal.
	// Uses context.Background() — request context may cancel before goroutine runs.
	go func(ctx context.Context, id uuid.UUID) {
		if err := h.listings.IncrementViewCount(ctx, id); err != nil {
			slog.Warn("failed to increment view count", "listing_id", id, "error", err)
		}
	}(context.Background(), listing.ID)

	respond(w, http.StatusOK, listing)
}

// HandleNewListingPage serves the create listing form.
// Returns JSON stub — replaced with template in Session 010.
func (h *ListingHandler) HandleNewListingPage(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{"page": "new listing"})
}

// HandleCreateListing creates a new listing in draft status.
// ADR-004: title, property_type, price_mxn, municipality required.
func (h *ListingHandler) HandleCreateListing(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	if !user.CanManageListings() {
		respondError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	if err := r.ParseForm(); err != nil {
		respondError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	title        := strings.TrimSpace(r.FormValue("title"))
	propertyType := strings.TrimSpace(r.FormValue("property_type"))
	priceStr     := strings.TrimSpace(r.FormValue("price_mxn"))
	municipality := strings.TrimSpace(r.FormValue("municipality"))

	if title == "" || propertyType == "" || priceStr == "" || municipality == "" {
		respondError(w, http.StatusBadRequest, "title, property_type, price_mxn, and municipality are required")
		return
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil || price <= 0 {
		respondError(w, http.StatusBadRequest, "price_mxn must be a positive number")
		return
	}

	// Validate property type against canonical vocabulary — ADR-005
	pt := models.PropertyType(propertyType)
	switch pt {
	case models.TypeLand, models.TypeHouse, models.TypeRancho, models.TypeCommercial, models.TypeCabin:
		// valid
	default:
		respondError(w, http.StatusBadRequest, "invalid property_type")
		return
	}

	// Generate slug from title — collision-safe
	slug := generateSlug(title)
	slug, err = h.ensureUniqueSlug(r, slug)
	if err != nil {
		slog.Error("failed to generate unique slug", "title", title, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to create listing")
		return
	}

	input := repo.CreateInput{
		OwnerID:      user.ID,
		Title:        title,
		Slug:         slug,
		PropertyType: pt,
		PriceMXN:     price,
		Municipality: municipality,
	}

	listing, err := h.listings.Create(r.Context(), input)
	if err != nil {
		slog.Error("failed to create listing",
			"owner_id", user.ID,
			"title", title,
			"error", err,
		)
		respondError(w, http.StatusInternalServerError, "failed to create listing")
		return
	}

	slog.Info("listing created",
		"listing_id", listing.ID,
		"owner_id", user.ID,
		"slug", listing.Slug,
	)

	// Session 010: replace with http.Redirect(w, r, "/listings/"+listing.Slug+"/edit", http.StatusSeeOther)
	respond(w, http.StatusCreated, listing)
}

// HandleDashboard returns all listings owned by the current user.
func (h *ListingHandler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	listings, err := h.listings.ListByOwner(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to load dashboard listings", "user_id", user.ID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load dashboard")
		return
	}

	respond(w, http.StatusOK, map[string]any{
		"user":     user.ToSafe(),
		"listings": listings,
		"total":    len(listings),
	})
}

// HandleDeleteListing archives a listing (soft delete).
// Only the owner or admin can delete.
func (h *ListingHandler) HandleDeleteListing(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	slug := chi.URLParam(r, "slug")
	listing, err := h.listings.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			respondError(w, http.StatusNotFound, "listing not found")
			return
		}
		slog.Error("failed to get listing for delete", "slug", slug, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to delete listing")
		return
	}

	// Ownership check — only owner or admin can delete
	if !listing.IsOwnedBy(user.ID) && !user.IsAdmin() {
		respondError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	if err := h.listings.Archive(r.Context(), listing.ID); err != nil {
		slog.Error("failed to archive listing", "listing_id", listing.ID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to delete listing")
		return
	}

	slog.Info("listing archived", "listing_id", listing.ID, "owner_id", user.ID)

	// Session 010: replace with http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	respond(w, http.StatusOK, map[string]string{"message": "listing deleted"})
}

// generateSlug creates a URL-safe slug from a title.
// Lowercase, spaces to hyphens, strips special characters.
func generateSlug(title string) string {
	slug := strings.ToLower(strings.TrimSpace(title))

	var result strings.Builder
	for _, r := range slug {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r)
		case r >= '0' && r <= '9':
			result.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			result.WriteRune('-')
		}
	}

	// Collapse multiple hyphens
	s := result.String()
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	return strings.Trim(s, "-")
}

// ensureUniqueSlug appends a counter suffix until the slug is unique.
func (h *ListingHandler) ensureUniqueSlug(r *http.Request, base string) (string, error) {
	slug := base
	for i := 1; i <= 10; i++ {
		exists, err := h.listings.SlugExists(r.Context(), slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		slug = base + "-" + strconv.Itoa(i)
	}
	return "", errors.New("could not generate unique slug after 10 attempts")
}
