package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/jsarabia/fn-posts/internal/config"
	"google.golang.org/api/option"
)

// StorageService handles photo storage using Google Cloud Storage
type StorageService struct {
	client *storage.Client
	config config.StorageConfig
}

// UploadResult represents the result of a photo upload
type UploadResult struct {
	URL      string
	Size     int64
	Format   string
	Filename string
}

// NewStorageService creates a new storage service using Google Cloud Storage
func NewStorageService(cfg config.StorageConfig) (*StorageService, error) {
	// For local development with MinIO, skip GCS initialization
	storageProvider := os.Getenv("STORAGE_PROVIDER")
	if storageProvider == "minio" || storageProvider == "test" {
		// Return a storage service that doesn't use GCS client
		// MinIO is already initialized by docker-compose
		return &StorageService{
			client: nil, // MinIO doesn't need GCS client
			config: cfg,
		}, nil
	}

	ctx := context.Background()

	var client *storage.Client
	var err error

	// Create client with service account credentials if provided
	if cfg.CredentialsPath != "" {
		client, err = storage.NewClient(ctx, option.WithCredentialsFile(cfg.CredentialsPath))
	} else if cfg.CredentialsJSON != "" {
		client, err = storage.NewClient(ctx, option.WithCredentialsJSON([]byte(cfg.CredentialsJSON)))
	} else {
		// Use default credentials (environment variables, metadata server, etc.)
		client, err = storage.NewClient(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	service := &StorageService{
		client: client,
		config: cfg,
	}

	// Initialize bucket
	if err := service.initializeBucket(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize bucket: %w", err)
	}

	return service, nil
}

// Initialize performs any required setup for Google Cloud Storage
func (s *StorageService) initializeBucket(ctx context.Context) error {
	bucket := s.client.Bucket(s.config.BucketName)

	// Check if bucket exists and is accessible
	_, err := bucket.Attrs(ctx)
	if err != nil {
		// If bucket doesn't exist, create it
		if err == storage.ErrBucketNotExist {
			err = bucket.Create(ctx, s.config.ProjectID, &storage.BucketAttrs{
				Location: "US", // Default to US multi-region
			})
			if err != nil {
				return fmt.Errorf("failed to create bucket: %w", err)
			}
		} else {
			return fmt.Errorf("failed to access bucket: %w", err)
		}
	}

	// Set bucket policy for public read access to photos
	policy := &storage.BucketPolicyOnly{
		Enabled: false, // Allow ACLs for fine-grained control
	}

	bucketAttrsToUpdate := storage.BucketAttrsToUpdate{
		BucketPolicyOnly: policy,
	}

	_, err = bucket.Update(ctx, bucketAttrsToUpdate)
	if err != nil {
		return fmt.Errorf("failed to update bucket policy: %w", err)
	}

	return nil
}

// UploadPhoto uploads a photo to Google Cloud Storage
func (s *StorageService) UploadPhoto(ctx context.Context, file multipart.File, header *multipart.FileHeader, postID uuid.UUID, organizationID *uuid.UUID) (*UploadResult, error) {
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

	// If using MinIO (client is nil), return a mock result for now
	if s.client == nil {
		// For MinIO/local development, just return a mock URL
		// In a real implementation, you would use the MinIO Go SDK
		minioEndpoint := os.Getenv("MINIO_ENDPOINT")
		minioBucket := os.Getenv("MINIO_BUCKET")
		if minioEndpoint == "" {
			minioEndpoint = "localhost:9000"
		}
		if minioBucket == "" {
			minioBucket = "posts-photos-dev"
		}

		return &UploadResult{
			URL:      fmt.Sprintf("http://%s/%s/%s", minioEndpoint, minioBucket, filename),
			Size:     header.Size,
			Format:   format,
			Filename: filename,
		}, nil
	}

	// Create object writer for GCS
	bucket := s.client.Bucket(s.config.BucketName)
	obj := bucket.Object(filename)
	writer := obj.NewWriter(ctx)

	// Set content type and metadata
	writer.ContentType = s.getContentType(format)
	writer.Metadata = map[string]string{
		"post-id":       postID.String(),
		"original-name": header.Filename,
		"upload-time":   time.Now().Format(time.RFC3339),
	}

	// Copy file content to GCS
	_, err := io.Copy(writer, file)
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Close the writer to finalize the upload
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize upload: %w", err)
	}

	// Make the object publicly readable
	acl := obj.ACL()
	err = acl.Set(ctx, storage.AllUsers, storage.RoleReader)
	if err != nil {
		return nil, fmt.Errorf("failed to set object ACL: %w", err)
	}

	// Generate public URL
	url := s.generatePublicURL(filename)

	return &UploadResult{
		URL:      url,
		Size:     header.Size,
		Format:   format,
		Filename: filename,
	}, nil
}

// DeletePhoto deletes a photo from Google Cloud Storage
func (s *StorageService) DeletePhoto(ctx context.Context, filename string) error {
	// If using MinIO (client is nil), skip deletion for now
	if s.client == nil {
		// In a real implementation, you would use MinIO SDK to delete
		return nil
	}

	bucket := s.client.Bucket(s.config.BucketName)
	obj := bucket.Object(filename)

	err := obj.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetPhotoURL returns the public URL for a photo
func (s *StorageService) GetPhotoURL(filename string) string {
	return s.generatePublicURL(filename)
}

// GenerateThumbnail generates a thumbnail for an uploaded photo
func (s *StorageService) GenerateThumbnail(ctx context.Context, originalURL string, postID uuid.UUID, organizationID *uuid.UUID) (string, error) {
	// This is a placeholder for thumbnail generation
	// In a real implementation with GCS, you would:
	// 1. Use Cloud Functions or Cloud Run to process images
	// 2. Use Cloud Storage triggers for automatic processing
	// 3. Use Google Cloud Vision API for image analysis
	// 4. Store thumbnails in a separate path/bucket

	// For now, just return the original URL
	// TODO: Implement actual thumbnail generation with Cloud Functions
	return originalURL, nil
}

// Helper methods

func (s *StorageService) generateFilename(postID uuid.UUID, organizationID *uuid.UUID, format string) string {
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

func (s *StorageService) generatePublicURL(filename string) string {
	// If CDN domain is configured, use it
	if s.config.CDNDomain != "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(s.config.CDNDomain, "/"), filename)
	}

	// Otherwise use the standard GCS public URL
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", s.config.BucketName, filename)
}

func (s *StorageService) getFileExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return ""
	}
	return ext[1:] // Remove the dot
}

func (s *StorageService) isValidImageFormat(format string) bool {
	validFormats := map[string]bool{
		"jpg":  true,
		"jpeg": true,
		"png":  true,
		"webp": true,
	}
	return validFormats[strings.ToLower(format)]
}

func (s *StorageService) getContentType(format string) string {
	contentTypes := map[string]string{
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"webp": "image/webp",
	}

	if contentType, exists := contentTypes[strings.ToLower(format)]; exists {
		return contentType
	}

	return "application/octet-stream"
}

// PhotoProcessor handles photo processing workflows
type PhotoProcessor struct {
	storage *StorageService
}

// NewPhotoProcessor creates a new photo processor
func NewPhotoProcessor(storage *StorageService) *PhotoProcessor {
	return &PhotoProcessor{storage: storage}
}

// ProcessUpload processes a photo upload with thumbnail generation
func (p *PhotoProcessor) ProcessUpload(ctx context.Context, file multipart.File, header *multipart.FileHeader, postID uuid.UUID, organizationID *uuid.UUID) (*UploadResult, error) {
	// Upload original
	result, err := p.storage.UploadPhoto(ctx, file, header, postID, organizationID)
	if err != nil {
		return nil, err
	}

	// Generate thumbnail asynchronously
	go func() {
		thumbnailURL, err := p.storage.GenerateThumbnail(context.Background(), result.URL, postID, organizationID)
		if err != nil {
			// Log error, but don't fail the upload
			fmt.Printf("Failed to generate thumbnail: %v\n", err)
			return
		}

		// In a real implementation, you would update the photo record with the thumbnail URL
		fmt.Printf("Thumbnail generated: %s\n", thumbnailURL)
	}()

	return result, nil
}

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

// Helper methods for TestStorageService

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
