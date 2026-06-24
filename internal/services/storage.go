// internal/services/storage.go
// Cloudflare R2 file upload and deletion.
// S3-compatible API via AWS SDK.
// Used for listing photos and profile avatars.

package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// StorageService handles file upload and deletion on Cloudflare R2.
type StorageService struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

// NewStorageService creates a StorageService connected to Cloudflare R2.
// Returns nil if credentials are empty — photo upload is optional at MVP.
func NewStorageService(accountID, accessKey, secretKey, bucket, publicURL string) *StorageService {
	if accountID == "" || accessKey == "" || secretKey == "" || bucket == "" {
		slog.Info("storage not configured — photo upload disabled")
		return nil
	}

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	})

	slog.Info("storage service initialized",
		"bucket", bucket,
		"endpoint", endpoint,
	)

	return &StorageService{
		client:    client,
		bucket:    bucket,
		publicURL: strings.TrimRight(publicURL, "/"),
	}
}

// Enabled returns true if the storage service is configured.
func (s *StorageService) Enabled() bool {
	return s != nil && s.client != nil
}

// UploadResult contains the result of a successful upload.
type UploadResult struct {
	StorageKey string // internal R2 object key
	PublicURL  string // public-facing URL for display
}

// Upload stores a file in R2 and returns the storage key and public URL.
// folder: "listings" or "avatars" — organizes files in R2
// filename: original filename from the upload
// contentType: MIME type (e.g. "image/jpeg")
// body: file contents
func (s *StorageService) Upload(ctx context.Context, folder, filename, contentType string, body io.Reader) (*UploadResult, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("storage: service not configured")
	}

	// Generate unique key: folder/timestamp-randomhex-filename
	key := generateStorageKey(folder, filename)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("storage.Upload: %w", err)
	}

	publicURL := fmt.Sprintf("%s/%s", s.publicURL, key)

	slog.Info("file uploaded",
		"key", key,
		"content_type", contentType,
		"public_url", publicURL,
	)

	return &UploadResult{
		StorageKey: key,
		PublicURL:  publicURL,
	}, nil
}

// Delete removes a file from R2 by its storage key.
func (s *StorageService) Delete(ctx context.Context, storageKey string) error {
	if !s.Enabled() {
		return fmt.Errorf("storage: service not configured")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(storageKey),
	})
	if err != nil {
		return fmt.Errorf("storage.Delete: %w", err)
	}

	slog.Info("file deleted", "key", storageKey)
	return nil
}

// PublicURL returns the full public URL for a storage key.
func (s *StorageService) PublicURL(storageKey string) string {
	return fmt.Sprintf("%s/%s", s.publicURL, storageKey)
}

// generateStorageKey creates a unique, collision-safe key.
// Format: folder/20260623-a1b2c3d4-originalname.jpg
func generateStorageKey(folder, filename string) string {
	// Clean filename
	ext := strings.ToLower(path.Ext(filename))
	if ext == "" {
		ext = ".bin"
	}

	// Generate random hex suffix
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	randHex := hex.EncodeToString(b)

	// Date prefix for easy browsing in R2 console
	datePrefix := time.Now().Format("20060102")

	// Clean base name
	baseName := strings.TrimSuffix(filename, ext)
	baseName = sanitizeFilename(baseName)
	if len(baseName) > 40 {
		baseName = baseName[:40]
	}

	return fmt.Sprintf("%s/%s-%s-%s%s", folder, datePrefix, randHex, baseName, ext)
}

// sanitizeFilename removes special characters from a filename.
func sanitizeFilename(name string) string {
	var result strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r)
		case r >= '0' && r <= '9':
			result.WriteRune(r)
		case r == '-' || r == '_':
			result.WriteRune(r)
		case r == ' ':
			result.WriteRune('-')
		}
	}
	return result.String()
}

// AllowedImageType returns true if the content type is a safe image format.
func AllowedImageType(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/jpg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

// MaxImageSize is the maximum file size for uploaded images (10MB).
const MaxImageSize = 10 << 20
