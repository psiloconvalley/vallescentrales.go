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

	"vallescentrales/internal/catalog"
	"vallescentrales/internal/middleware"
	"vallescentrales/internal/models"
	"vallescentrales/internal/repo"
)

// ListingHandler handles all listing HTTP endpoints.
type ListingHandler struct {
	listings *repo.ListingRepo
	render   Renderer
}

// NewListingHandler creates a ListingHandler.
func NewListingHandler(listings *repo.ListingRepo, render Renderer) *ListingHandler {
	return &ListingHandler{
		listings: listings,
		render:   render,
	}
}

// pageData builds the standard template data map.
func (h *ListingHandler) pageData(r *http.Request, title string, extra map[string]any) map[string]any {
	data := map[string]any{
		"Meta": map[string]string{
			"Title":       title,
			"Description": "Tierra y casas en los Valles Centrales de Oaxaca",
		},
		"User":      middleware.UserFromContext(r.Context()),
		"Flash":     nil,
		"CSRFToken": middleware.CSRFToken(r),
	}
	for k, v := range extra {
		data[k] = v
	}
	return data
}

// HandleHome serves the homepage with featured listings.
func (h *ListingHandler) HandleHome(w http.ResponseWriter, r *http.Request) {
	filter := repo.ListFilter{Page: 1, PageSize: 6}
	listings, _, err := h.listings.List(r.Context(), filter)
	if err != nil {
		slog.Error("failed to load homepage listings", "error", err)
		listings = nil
	}

	h.render.Render(w, r, "home.tmpl", h.pageData(r, "", map[string]any{
		"Listings": listings,
	}))
}

// HandleListListings returns active listings with optional filters.
func (h *ListingHandler) HandleListListings(w http.ResponseWriter, r *http.Request) {
	filter := repo.ListFilter{Page: 1, PageSize: 20}

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

	user := middleware.UserFromContext(r.Context())
	if !listing.IsPublic() && (user == nil || !listing.IsOwnedBy(user.ID)) {
		respondError(w, http.StatusNotFound, "listing not found")
		return
	}

	media, err := h.listings.GetMedia(r.Context(), listing.ID)
	if err != nil {
		slog.Warn("failed to load listing media", "listing_id", listing.ID, "error", err)
	} else {
		listing.Media = media
	}

	go func(ctx context.Context, id uuid.UUID) {
		if err := h.listings.IncrementViewCount(ctx, id); err != nil {
			slog.Warn("failed to increment view count", "listing_id", id, "error", err)
		}
	}(context.Background(), listing.ID)

	respond(w, http.StatusOK, listing)
}

// ─── Create Listing ──────────────────────────────────────────────────────────

// listingFormData holds form values for re-rendering on errors.
type listingFormData struct {
	Title          string
	PropertyType   string
	PriceMXN       string
	Municipality   string
	Community      string
	Description    string
	AreaM2         string
	ConstructionM2 string
}

// HandleNewListingPage serves the create listing form.
func (h *ListingHandler) HandleNewListingPage(w http.ResponseWriter, r *http.Request) {
	h.render.Render(w, r, "listing_new.tmpl", h.pageData(r, "Publicar Propiedad", map[string]any{
		"Regions":        catalog.Regions(),
		"Municipalities": catalog.MunicipalitiesByRegion(),
	}))
}

// renderListingForm re-renders the form with error and preserved data.
func (h *ListingHandler) renderListingForm(w http.ResponseWriter, r *http.Request, errMsg string, fd listingFormData) {
	h.render.Render(w, r, "listing_new.tmpl", h.pageData(r, "Publicar Propiedad", map[string]any{
		"Error":          errMsg,
		"FormData":       fd,
		"Regions":        catalog.Regions(),
		"Municipalities": catalog.MunicipalitiesByRegion(),
	}))
}

// HandleCreateListing creates a new listing in draft status.
func (h *ListingHandler) HandleCreateListing(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	if !user.CanManageListings() {
		respondError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderListingForm(w, r, "Datos de formulario inválidos", listingFormData{})
		return
	}

	title          := strings.TrimSpace(r.FormValue("title"))
	propertyType   := strings.TrimSpace(r.FormValue("property_type"))
	priceStr       := strings.TrimSpace(r.FormValue("price_mxn"))
	municipality   := strings.TrimSpace(r.FormValue("municipality"))
	community      := strings.TrimSpace(r.FormValue("community"))
	description    := strings.TrimSpace(r.FormValue("description"))
	areaStr        := strings.TrimSpace(r.FormValue("area_m2"))
	constructionStr := strings.TrimSpace(r.FormValue("construction_m2"))

	fd := listingFormData{
		Title:          title,
		PropertyType:   propertyType,
		PriceMXN:       priceStr,
		Municipality:   municipality,
		Community:      community,
		Description:    description,
		AreaM2:         areaStr,
		ConstructionM2: constructionStr,
	}

	// Validate required fields
	if title == "" || propertyType == "" || priceStr == "" || municipality == "" {
		h.renderListingForm(w, r, "Título, tipo, precio y municipio son obligatorios", fd)
		return
	}

	if len(title) < 10 {
		h.renderListingForm(w, r, "El título debe tener al menos 10 caracteres", fd)
		return
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil || price <= 0 {
		h.renderListingForm(w, r, "El precio debe ser un número positivo", fd)
		return
	}

	pt := models.PropertyType(propertyType)
	switch pt {
	case models.TypeLand, models.TypeHouse, models.TypeRancho, models.TypeCommercial, models.TypeCabin:
	default:
		h.renderListingForm(w, r, "Tipo de propiedad inválido", fd)
		return
	}

	slug := generateSlug(title)
	slug, err = h.ensureUniqueSlug(r, slug)
	if err != nil {
		slog.Error("failed to generate unique slug", "title", title, "error", err)
		h.renderListingForm(w, r, "Error al crear la propiedad", fd)
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
		h.renderListingForm(w, r, "Error al crear la propiedad", fd)
		return
	}

	slog.Info("listing created",
		"listing_id", listing.ID,
		"owner_id", user.ID,
		"slug", listing.Slug,
	)

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// ─── Dashboard ───────────────────────────────────────────────────────────────

// HandleDashboard renders the owner dashboard with their listings.
func (h *ListingHandler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	listings, err := h.listings.ListByOwner(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to load dashboard listings", "user_id", user.ID, "error", err)
		listings = nil
	}

	h.render.Render(w, r, "dashboard.tmpl", h.pageData(r, "Mi Panel", map[string]any{
		"Listings": listings,
	}))
}

// ─── Delete ──────────────────────────────────────────────────────────────────

// HandleDeleteListing archives a listing (soft delete).
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
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

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

	s := result.String()
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	return strings.Trim(s, "-")
}

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
