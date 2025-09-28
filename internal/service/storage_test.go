package service

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jsarabia/fn-posts/internal/config"
)

// TestStorageService provides a simple storage implementation for testing
type TestStorageService struct {
	config config.StorageConfig
	files  map[string][]byte // In-memory file storage for testing
}

// NewTestStorageService creates a new test storage service
func NewTestStorageService(cfg config.StorageConfig) *TestStorageService {
	return &TestStorageService{
		config: cfg,
		files:  make(map[string][]byte),
	}
}

// UploadPhoto uploads a photo to test storage (in-memory)
func (s *TestStorageService) UploadPhoto(ctx context.Context, file multipart.File, header *multipart.FileHeader, postID uuid.UUID, organizationID *uuid.UUID) (*UploadResult, error) {
	// Validate file format
	format := s.getFileExtension(header.Filename)
	if !s.isValidImageFormat(format) {
		return nil, fmt.Errorf("invalid image format: %s", format)
	}

	// Validate file size (max 10MB)
	if header.Size > 10*1024*1024 {
		return nil, fmt.Errorf("file too large: %d bytes (max 10MB)", header.Size)
	}

	// Generate unique filename
	filename := s.generateFilename(postID, organizationID, format)

	// Read file content (for testing, we'll just store the filename)
	s.files[filename] = []byte(fmt.Sprintf("test-content-%s", filename))

	// Generate public URL for testing
	url := s.generatePublicURL(filename)

	return &UploadResult{
		URL:      url,
		Size:     header.Size,
		Format:   format,
		Filename: filename,
	}, nil
}

// DeletePhoto deletes a photo from test storage
func (s *TestStorageService) DeletePhoto(ctx context.Context, filename string) error {
	if _, exists := s.files[filename]; !exists {
		return fmt.Errorf("file not found: %s", filename)
	}
	delete(s.files, filename)
	return nil
}

// GetPhotoURL returns the public URL for a photo
func (s *TestStorageService) GetPhotoURL(filename string) string {
	return s.generatePublicURL(filename)
}

// GenerateThumbnail generates a thumbnail (test implementation)
func (s *TestStorageService) GenerateThumbnail(ctx context.Context, originalURL string, postID uuid.UUID, organizationID *uuid.UUID) (string, error) {
	// For testing, just return a modified URL
	return strings.Replace(originalURL, "original", "thumbnail", 1), nil
}

// Helper methods (copied from storage.go)

func (s *TestStorageService) generateFilename(postID uuid.UUID, organizationID *uuid.UUID, format string) string {
	timestamp := time.Now().Format("20060102150405")
	uniqueID := uuid.New().String()[:8]

	var path string
	if organizationID != nil {
		path = fmt.Sprintf("%s/%s/original/%s_%s.%s",
			organizationID.String(),
			postID.String(),
			timestamp,
			uniqueID,
			format)
	} else {
		path = fmt.Sprintf("public/%s/original/%s_%s.%s",
			postID.String(),
			timestamp,
			uniqueID,
			format)
	}

	return path
}

func (s *TestStorageService) generatePublicURL(filename string) string {
	// For testing, use a test URL
	return fmt.Sprintf("http://localhost:9001/posts-test-bucket/%s", filename)
}

func (s *TestStorageService) getFileExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return ""
	}
	return ext[1:] // Remove the dot
}

func (s *TestStorageService) isValidImageFormat(format string) bool {
	validFormats := map[string]bool{
		"jpg":  true,
		"jpeg": true,
		"png":  true,
		"webp": true,
	}
	return validFormats[strings.ToLower(format)]
}
