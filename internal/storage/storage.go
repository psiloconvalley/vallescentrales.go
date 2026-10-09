// Package storage abstracts image persistence. The interface lets us
// swap LocalStorage (dev) for R2Storage (prod) without touching callers.
package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrFileTooLarge     = errors.New("el archivo excede el tamaño máximo permitido (10 MB)")
	ErrInvalidImageType = errors.New("formato no permitido (solo JPEG, PNG o WebP)")
	ErrTooManyImages    = errors.New("máximo 10 imágenes por propiedad")
	ErrEmptyFile        = errors.New("el archivo está vacío")
)

const (
	MaxFileSize = 10 << 20 // 10 MB per file
	MaxFiles    = 10
)

var allowedMIME = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// Service abstracts image persistence.
type Service interface {
	Upload(ctx context.Context, fh *multipart.FileHeader) (string, error)
	Delete(ctx context.Context, url string) error
}

// LocalStorage writes uploads to disk under static/uploads for development.
type LocalStorage struct {
	UploadDir string
	URLPrefix string
}

// NewLocalStorage creates and ensures the upload directory exists.
func NewLocalStorage(uploadDir string) (*LocalStorage, error) {
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &LocalStorage{
		UploadDir: uploadDir,
		URLPrefix: "/static/uploads",
	}, nil
}

// Upload validates magic bytes, generates a safe filename, and writes to disk.
func (l *LocalStorage) Upload(ctx context.Context, fh *multipart.FileHeader) (string, error) {
	if fh.Size == 0 {
		return "", ErrEmptyFile
	}
	if fh.Size > MaxFileSize {
		return "", ErrFileTooLarge
	}

	src, err := fh.Open()
	if err != nil {
		return "", fmt.Errorf("open upload: %w", err)
	}
	defer src.Close()

	// Sniff first 512 bytes to validate real MIME (never trust extension)
	head := make([]byte, 512)
	n, err := src.Read(head)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read header: %w", err)
	}
	mime := http.DetectContentType(head[:n])
	ext, ok := allowedMIME[mime]
	if !ok {
		return "", ErrInvalidImageType
	}

	// Rewind
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("rewind: %w", err)
	}

	// Generate unguessable filename
	rnd := make([]byte, 8)
	if _, err := rand.Read(rnd); err != nil {
		return "", fmt.Errorf("entropy: %w", err)
	}
	filename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), hex.EncodeToString(rnd), ext)
	dstPath := filepath.Join(l.UploadDir, filename)

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("create dest: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		_ = os.Remove(dstPath)
		return "", fmt.Errorf("write dest: %w", err)
	}

	return l.URLPrefix + "/" + filename, nil
}

// Delete removes an uploaded file. Path traversal is rejected.
func (l *LocalStorage) Delete(ctx context.Context, fileURL string) error {
	filename := filepath.Base(fileURL)
	if strings.Contains(filename, "..") || filename == "" || filename == "." {
		return errors.New("invalid filename")
	}
	fullPath := filepath.Join(l.UploadDir, filename)
	err := os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
