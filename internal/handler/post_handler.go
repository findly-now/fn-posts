package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jsarabia/fn-posts/internal/domain"
	"github.com/jsarabia/fn-posts/internal/service"
)

type PostHandler struct {
	postService *service.PostService
}

func NewPostHandler(postService *service.PostService) *PostHandler {
	return &PostHandler{
		postService: postService,
	}
}

type CreatePostRequest struct {
	Title          string                 `json:"title" binding:"required,min=1,max=200"`
	Description    string                 `json:"description" binding:"max=2000"`
	Location       domain.Location        `json:"location" binding:"required"`
	RadiusMeters   int                    `json:"radius_meters" binding:"min=100,max=50000"`
	Type           domain.PostType        `json:"type" binding:"required"`
	OrganizationID *domain.OrganizationID `json:"organization_id,omitempty"`
}

type UpdatePostRequest struct {
	Title       string `json:"title" binding:"required,min=1,max=200"`
	Description string `json:"description" binding:"max=2000"`
}

type UpdatePostStatusRequest struct {
	Status domain.PostStatus `json:"status" binding:"required"`
}

type PostResponse struct {
	ID             uuid.UUID         `json:"id"`
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	Photos         []PhotoResponse   `json:"photos"`
	Location       domain.Location   `json:"location"`
	RadiusMeters   int               `json:"radius_meters"`
	Status         domain.PostStatus `json:"status"`
	Type           domain.PostType   `json:"type"`
	CreatedBy      uuid.UUID         `json:"created_by"`
	OrganizationID *uuid.UUID        `json:"organization_id,omitempty"`
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
}

type PhotoResponse struct {
	ID           uuid.UUID `json:"id"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	Caption      string    `json:"caption,omitempty"`
	DisplayOrder int       `json:"display_order"`
	CreatedAt    string    `json:"created_at"`
}

type ListPostsResponse struct {
	Posts  []PostResponse `json:"posts"`
	Total  int64          `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

func (h *PostHandler) CreatePost(c *gin.Context) {
	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := h.getUserIDFromContext(c)
	if userID.IsZero() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	post, err := h.postService.CreatePost(
		c.Request.Context(),
		req.Title,
		req.Description,
		req.Location,
		req.RadiusMeters,
		req.Type,
		userID,
		req.OrganizationID,
	)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, h.toPostResponse(post))
}

func (h *PostHandler) GetPost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := domain.PostIDFromString(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	post, err := h.postService.GetPostByID(c.Request.Context(), id)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, h.toPostResponse(post))
}

func (h *PostHandler) UpdatePost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := domain.PostIDFromString(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	var req UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	post, err := h.postService.UpdatePost(c.Request.Context(), id, req.Title, req.Description)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, h.toPostResponse(post))
}

func (h *PostHandler) UpdatePostStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := domain.PostIDFromString(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	var req UpdatePostStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	post, err := h.postService.UpdatePostStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.toPostResponse(post))
}

func (h *PostHandler) DeletePost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := domain.PostIDFromString(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	if err := h.postService.DeletePost(c.Request.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete post"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *PostHandler) ListPosts(c *gin.Context) {
	filters := h.parseFiltersFromQuery(c)

	posts, err := h.postService.ListPosts(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list posts"})
		return
	}

	total, err := h.postService.CountPosts(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count posts"})
		return
	}

	response := ListPostsResponse{
		Posts:  h.toPostResponses(posts),
		Total:  total,
		Limit:  filters.Limit,
		Offset: filters.Offset,
	}

	c.JSON(http.StatusOK, response)
}

func (h *PostHandler) SearchNearbyPosts(c *gin.Context) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	radiusStr := c.Query("radius")

	if latStr == "" || lngStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat and lng parameters are required"})
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid latitude"})
		return
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid longitude"})
		return
	}

	location, err := domain.NewLocation(lat, lng)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	radius := 1000 // Default 1km
	if radiusStr != "" {
		if r, err := strconv.Atoi(radiusStr); err == nil {
			radius = r
		}
	}

	var postType *domain.PostType
	if typeStr := c.Query("type"); typeStr != "" {
		pt := domain.PostType(typeStr)
		postType = &pt
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	posts, err := h.postService.SearchNearbyPosts(c.Request.Context(), location, radius, postType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search nearby posts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"posts":  h.toPostResponses(posts),
		"count":  len(posts),
		"limit":  limit,
		"offset": offset,
	})
}

func (h *PostHandler) GetUserPosts(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := domain.UserIDFromString(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	posts, err := h.postService.GetPostsByUser(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user posts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"posts":  h.toPostResponses(posts),
		"count":  len(posts),
		"limit":  limit,
		"offset": offset,
	})
}

func (h *PostHandler) parseFiltersFromQuery(c *gin.Context) domain.PostFilters {
	filters := domain.PostFilters{}

	if status := c.Query("status"); status != "" {
		s := domain.PostStatus(status)
		filters.Status = &s
	}

	if postType := c.Query("type"); postType != "" {
		t := domain.PostType(postType)
		filters.Type = &t
	}

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := domain.UserIDFromString(userIDStr); err == nil {
			filters.UserID = &userID
		}
	}

	if orgIDStr := c.Query("organization_id"); orgIDStr != "" {
		if orgID, err := domain.OrganizationIDFromString(orgIDStr); err == nil {
			filters.OrganizationID = &orgID
		}
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters.Limit = l
		}
	}

	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filters.Offset = o
		}
	}

	return filters
}

func (h *PostHandler) toPostResponse(post *domain.Post) PostResponse {
	photos := make([]PhotoResponse, len(post.Photos()))
	for i, photo := range post.Photos() {
		photos[i] = PhotoResponse{
			ID:           photo.ID().UUID(),
			URL:          photo.URL(),
			ThumbnailURL: photo.ThumbnailURL(),
			Caption:      photo.Caption(),
			DisplayOrder: photo.DisplayOrder(),
			CreatedAt:    photo.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	var orgID *uuid.UUID
	if post.OrganizationID() != nil {
		id := post.OrganizationID().UUID()
		orgID = &id
	}

	return PostResponse{
		ID:             post.ID().UUID(),
		Title:          post.Title(),
		Description:    post.Description(),
		Photos:         photos,
		Location:       post.Location(),
		RadiusMeters:   post.RadiusMeters(),
		Status:         post.Status(),
		Type:           post.PostType(),
		CreatedBy:      post.CreatedBy().UUID(),
		OrganizationID: orgID,
		CreatedAt:      post.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      post.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (h *PostHandler) toPostResponses(posts []*domain.Post) []PostResponse {
	responses := make([]PostResponse, len(posts))
	for i, post := range posts {
		responses[i] = h.toPostResponse(post)
	}
	return responses
}

func (h *PostHandler) getUserIDFromContext(c *gin.Context) domain.UserID {
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
