package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jsarabia/fn-posts/internal/domain"
	"github.com/jsarabia/fn-posts/internal/service"
)

type PhotoHandler struct {
	postService *service.PostService
	storage     StorageInterface
}

func NewPhotoHandler(postService *service.PostService, storage StorageInterface) *PhotoHandler {
	return &PhotoHandler{
		postService: postService,
		storage:     storage,
	}
}

type UploadPhotoResponse struct {
	Photo PhotoResponse `json:"photo"`
	URL   string        `json:"url"`
}

func (h *PhotoHandler) UploadPhoto(c *gin.Context) {
	postIDStr := c.Param("postId")
	postID, err := domain.PostIDFromString(postIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	// Get multipart form
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	files := form.File["photos"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No photos provided"})
		return
	}

	if len(files) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 10 photos allowed"})
		return
	}

	userID := h.getUserIDFromContext(c)
	if userID.IsZero() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get organization ID from form or context
	var organizationID *domain.OrganizationID
	if orgIDStr := c.PostForm("organization_id"); orgIDStr != "" {
		if orgID, err := domain.OrganizationIDFromString(orgIDStr); err == nil {
			organizationID = &orgID
		}
	}

	var uploadedPhotos []UploadPhotoResponse
	var errors []string

	for i, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to open file %s: %v", fileHeader.Filename, err))
			continue
		}
		defer file.Close()

		// Process upload
		var orgIDPtr *uuid.UUID
		if organizationID != nil {
			orgID := organizationID.UUID()
			orgIDPtr = &orgID
		}
		result, err := h.storage.UploadPhoto(c.Request.Context(), file, fileHeader, postID.UUID(), orgIDPtr)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to upload %s: %v", fileHeader.Filename, err))
			continue
		}

		// Create photo domain object
		photoReq := domain.CreatePhotoRequest{
			PostID:       postID,
			URL:          result.URL,
			Caption:      c.PostForm(fmt.Sprintf("caption_%d", i)),
			DisplayOrder: i + 1,
			Format:       result.Format,
			SizeBytes:    result.Size,
		}

		photo, err := h.postService.AddPhotoToPost(c.Request.Context(), postID, photoReq)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to save photo %s: %v", fileHeader.Filename, err))
			// Clean up uploaded file
			h.storage.DeletePhoto(c.Request.Context(), result.Filename)
			continue
		}

		uploadedPhotos = append(uploadedPhotos, UploadPhotoResponse{
			Photo: PhotoResponse{
				ID:           photo.ID().UUID(),
				URL:          photo.URL(),
				ThumbnailURL: photo.ThumbnailURL(),
				Caption:      photo.Caption(),
				DisplayOrder: photo.DisplayOrder(),
				CreatedAt:    photo.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
			},
			URL: result.URL,
		})
	}

	response := gin.H{
		"uploaded_photos": uploadedPhotos,
		"success_count":   len(uploadedPhotos),
		"total_count":     len(files),
	}

	if len(errors) > 0 {
		response["errors"] = errors
		c.JSON(http.StatusPartialContent, response)
	} else {
		c.JSON(http.StatusCreated, response)
	}
}

func (h *PhotoHandler) DeletePhoto(c *gin.Context) {
	postIDStr := c.Param("postId")
	postID, err := domain.PostIDFromString(postIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	photoIDStr := c.Param("photoId")
	photoID, err := domain.PhotoIDFromString(photoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid photo ID"})
		return
	}

	err = h.postService.RemovePhotoFromPost(c.Request.Context(), postID, photoID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
			return
		}
		if strings.Contains(err.Error(), "cannot remove last photo") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot remove the last photo from a post"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete photo"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *PhotoHandler) getUserIDFromContext(c *gin.Context) domain.UserID {
	// This would typically be set by authentication middleware
	// For now, return a dummy UUID or get from header
	if userIDStr := c.GetHeader("X-User-ID"); userIDStr != "" {
		if userID, err := domain.UserIDFromString(userIDStr); err == nil {
			return userID
		}
	}

	// In a real implementation, this would be extracted from JWT token
	return domain.NewUserID() // Temporary for development
}
