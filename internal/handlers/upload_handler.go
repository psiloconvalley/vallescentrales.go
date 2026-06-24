// internal/handlers/upload_handler.go
// Handles file uploads for listing photos and profile avatars.
// Rule 42: handlers = HTTP only. No SQL. No business logic.

package handlers

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"vallescentrales/internal/middleware"
	"vallescentrales/internal/repo"
	"vallescentrales/internal/services"
)

// UploadHandler handles all file upload operations.
type UploadHandler struct {
	storage  *services.StorageService
	listings *repo.ListingRepo
	users    *repo.UserRepo
}

// NewUploadHandler creates an UploadHandler.
func NewUploadHandler(storage *services.StorageService, listings *repo.ListingRepo, users *repo.UserRepo) *UploadHandler {
	return &UploadHandler{
		storage:  storage,
		listings: listings,
		users:    users,
	}
}

// HandleUploadListingPhotos handles photo upload for a listing.
// POST /listings/{slug}/photos
func (h *UploadHandler) HandleUploadListingPhotos(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	if !h.storage.Enabled() {
		respondError(w, http.StatusServiceUnavailable, "photo upload not available")
		return
	}

	slug := chi.URLParam(r, "slug")
	listing, err := h.listings.GetBySlug(r.Context(), slug)
	if err != nil {
		respondError(w, http.StatusNotFound, "listing not found")
		return
	}

	if !listing.IsOwnedBy(user.ID) && !user.IsAdmin() {
		respondError(w, http.StatusForbidden, "not authorized")
		return
	}

	// 50MB total multipart form limit
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		slog.Warn("failed to parse multipart form", "error", err)
		respondError(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}

	files := r.MultipartForm.File["photos"]
	if len(files) == 0 {
		respondError(w, http.StatusBadRequest, "no files uploaded")
		return
	}

	if len(files) > 20 {
		respondError(w, http.StatusBadRequest, "maximum 20 photos per upload")
		return
	}

	existingMedia, _ := h.listings.GetMedia(r.Context(), listing.ID)
	sortOrder := len(existingMedia)
	firstPhotoPrimary := sortOrder == 0

	var uploaded int

	for _, fileHeader := range files {
		if fileHeader.Size > services.MaxImageSize {
			slog.Warn("file too large", "filename", fileHeader.Filename, "size", fileHeader.Size)
			continue
		}

		contentType := fileHeader.Header.Get("Content-Type")
		if !services.AllowedImageType(contentType) {
			slog.Warn("invalid image type", "filename", fileHeader.Filename, "type", contentType)
			continue
		}

		file, err := fileHeader.Open()
		if err != nil {
			slog.Error("failed to open uploaded file", "filename", fileHeader.Filename, "error", err)
			continue
		}

		result, err := h.storage.Upload(r.Context(), "listings", fileHeader.Filename, contentType, file)
		_ = file.Close()
		if err != nil {
			slog.Error("failed to upload to R2", "filename", fileHeader.Filename, "error", err)
			continue
		}

		_, err = h.listings.AddMedia(
			r.Context(),
			listing.ID,
			result.PublicURL,
			result.StorageKey,
			firstPhotoPrimary && uploaded == 0,
			sortOrder,
		)
		if err != nil {
			slog.Error("failed to save media record", "listing_id", listing.ID, "error", err)
			_ = h.storage.Delete(r.Context(), result.StorageKey)
			continue
		}

		uploaded++
		sortOrder++
	}

	if uploaded == 0 {
		respondError(w, http.StatusBadRequest, "no valid photos uploaded")
		return
	}

	slog.Info("photos uploaded",
		"listing_id", listing.ID,
		"count", uploaded,
		"owner_id", user.ID,
	)

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// HandleUploadAvatar handles profile avatar upload.
// POST /dashboard/avatar
func (h *UploadHandler) HandleUploadAvatar(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	if !h.storage.Enabled() {
		respondError(w, http.StatusServiceUnavailable, "avatar upload not available")
		return
	}

	// 5MB multipart limit for avatars
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		respondError(w, http.StatusBadRequest, "no file uploaded")
		return
	}
	defer file.Close()

	if header.Size > services.MaxImageSize {
		respondError(w, http.StatusBadRequest, "image must be under 10MB")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if !services.AllowedImageType(contentType) {
		respondError(w, http.StatusBadRequest, "only JPEG, PNG, and WebP images are allowed")
		return
	}

	result, err := h.storage.Upload(r.Context(), "avatars", header.Filename, contentType, file)
	if err != nil {
		slog.Error("failed to upload avatar", "user_id", user.ID, "error", err)
		respondError(w, http.StatusInternalServerError, "upload failed")
		return
	}

	_, err = h.users.UpdateAvatar(r.Context(), user.ID, result.PublicURL)
	if err != nil {
		slog.Error("failed to update avatar URL", "user_id", user.ID, "error", err)
		_ = h.storage.Delete(r.Context(), result.StorageKey)
		respondError(w, http.StatusInternalServerError, "upload failed")
		return
	}

	slog.Info("avatar uploaded", "user_id", user.ID, "url", result.PublicURL)

	http.Redirect(w, r, "/dashboard/profile", http.StatusSeeOther)
}
