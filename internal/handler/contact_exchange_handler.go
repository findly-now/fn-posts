package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/jsarabia/fn-posts/internal/domain"
	"github.com/jsarabia/fn-posts/internal/service"
)

type ContactExchangeHandler struct {
	contactExchangeService *service.ContactExchangeService
}

func NewContactExchangeHandler(contactExchangeService *service.ContactExchangeService) *ContactExchangeHandler {
	return &ContactExchangeHandler{
		contactExchangeService: contactExchangeService,
	}
}

// RegisterRoutes registers contact exchange routes
func (h *ContactExchangeHandler) RegisterRoutes(router *gin.RouterGroup) {
	contacts := router.Group("/contacts")
	{
		contacts.POST("/exchange", h.CreateContactExchangeRequest)
		contacts.GET("/exchange/:id", h.GetContactExchangeRequest)
		contacts.POST("/exchange/:id/approve", h.ApproveContactExchange)
		contacts.POST("/exchange/:id/deny", h.DenyContactExchange)
		contacts.DELETE("/exchange/:id", h.CancelContactExchange)
		contacts.GET("/exchange", h.ListContactExchangeRequests)
	}
}

type CreateContactExchangeRequestDTO struct {
	PostID               string                        `json:"post_id" binding:"required"`
	Message              *string                       `json:"message,omitempty"`
	VerificationRequired bool                          `json:"verification_required"`
	VerificationDetails  *VerificationDetailsDTO       `json:"verification_details,omitempty"`
	ExpirationHours      int                           `json:"expiration_hours,omitempty"`
}

type VerificationDetailsDTO struct {
	Method       string   `json:"method" binding:"required"`
	Question     *string  `json:"question,omitempty"`
	Requirements []string `json:"requirements,omitempty"`
}

type ApproveContactExchangeRequestDTO struct {
	ApprovalType string         `json:"approval_type" binding:"required"`
	ContactInfo  *ContactInfoDTO `json:"contact_info,omitempty"`
}

type ContactInfoDTO struct {
	Email           *string                  `json:"email,omitempty"`
	Phone           *string                  `json:"phone,omitempty"`
	PreferredMethod string                   `json:"preferred_method" binding:"required"`
	Message         *string                  `json:"message,omitempty"`
	Restrictions    *SharingRestrictionsDTO  `json:"restrictions,omitempty"`
}

type EncryptedContactInfoDTO struct {
	Email               *string                  `json:"email,omitempty"`
	Phone               *string                  `json:"phone,omitempty"`
	PreferredMethod     string                   `json:"preferred_method" binding:"required"`
	Message             *string                  `json:"message,omitempty"`
	SharingRestrictions *SharingRestrictionsDTO  `json:"sharing_restrictions,omitempty"`
}

type SharingRestrictionsDTO struct {
	ExpiresAfterHours int  `json:"expires_after_hours"`
	SingleUse         bool `json:"single_use"`
	PlatformMediated  bool `json:"platform_mediated"`
}

type DenyContactExchangeRequestDTO struct {
	DenialReason  string  `json:"denial_reason" binding:"required"`
	DenialMessage *string `json:"denial_message,omitempty"`
}

type ContactExchangeResponseDTO struct {
	ID                   string                      `json:"id"`
	PostID               string                      `json:"post_id"`
	RequesterUserID      string                      `json:"requester_user_id"`
	OwnerUserID          string                      `json:"owner_user_id"`
	Status               string                      `json:"status"`
	Message              *string                     `json:"message,omitempty"`
	VerificationRequired bool                        `json:"verification_required"`
	VerificationDetails  *VerificationDetailsDTO     `json:"verification_details,omitempty"`
	ApprovalType         *string                     `json:"approval_type,omitempty"`
	DenialReason         *string                     `json:"denial_reason,omitempty"`
	DenialMessage        *string                     `json:"denial_message,omitempty"`
	EncryptedContactInfo *EncryptedContactInfoDTO    `json:"contact_info,omitempty"`
	ExpiresAt            string                      `json:"expires_at"`
	CreatedAt            string                      `json:"created_at"`
	UpdatedAt            string                      `json:"updated_at"`
}

// CreateContactExchangeRequest creates a new contact exchange request
func (h *ContactExchangeHandler) CreateContactExchangeRequest(c *gin.Context) {
	var req CreateContactExchangeRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Get user ID from context (assumes authentication middleware sets this)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	requesterUserID, err := domain.UserIDFromString(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	postID, err := domain.PostIDFromString(req.PostID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	// Convert verification details
	var verificationDetails *domain.VerificationDetails
	if req.VerificationDetails != nil {
		verificationDetails = &domain.VerificationDetails{
			Method:       domain.VerificationMethod(req.VerificationDetails.Method),
			Question:     req.VerificationDetails.Question,
			Requirements: req.VerificationDetails.Requirements,
		}
	}

	// Set default expiration if not provided
	expirationHours := req.ExpirationHours
	if expirationHours <= 0 {
		expirationHours = 72 // Default 3 days
	}

	cmd := service.CreateContactExchangeCommand{
		PostID:               postID,
		RequesterUserID:      requesterUserID,
		Message:              req.Message,
		VerificationRequired: req.VerificationRequired,
		VerificationDetails:  verificationDetails,
		ExpirationHours:      expirationHours,
	}

	request, err := h.contactExchangeService.CreateContactExchangeRequest(c.Request.Context(), cmd)
	if err != nil {
		if domain.IsPostError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create contact exchange request"})
		return
	}

	response := h.toContactExchangeResponseDTO(request)
	c.JSON(http.StatusCreated, response)
}

// GetContactExchangeRequest retrieves a contact exchange request by ID
func (h *ContactExchangeHandler) GetContactExchangeRequest(c *gin.Context) {
	requestID, err := domain.ContactExchangeRequestIDFromString(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	request, err := h.contactExchangeService.GetContactExchangeRequest(c.Request.Context(), requestID)
	if err != nil {
		if domain.IsPostErrorCode(err, domain.ContactExchangeErrorNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contact exchange request not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve contact exchange request"})
		return
	}

	// Check if user is authorized to view this request
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userID, err := domain.UserIDFromString(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if !userID.Equals(request.RequesterUserID()) && !userID.Equals(request.OwnerUserID()) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to view this request"})
		return
	}

	response := h.toContactExchangeResponseDTO(request)
	c.JSON(http.StatusOK, response)
}

// ApproveContactExchange approves a contact exchange request
func (h *ContactExchangeHandler) ApproveContactExchange(c *gin.Context) {
	requestID, err := domain.ContactExchangeRequestIDFromString(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var req ApproveContactExchangeRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Get user ID from context
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userID, err := domain.UserIDFromString(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get the request to verify ownership
	request, err := h.contactExchangeService.GetContactExchangeRequest(c.Request.Context(), requestID)
	if err != nil {
		if domain.IsPostErrorCode(err, domain.ContactExchangeErrorNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contact exchange request not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve contact exchange request"})
		return
	}

	// Verify user is the owner
	if !userID.Equals(request.OwnerUserID()) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the post owner can approve contact exchange requests"})
		return
	}

	// Convert contact info
	var contactInfo *domain.ContactInfo
	if req.ContactInfo != nil {
		var restrictions *domain.SharingRestrictions
		if req.ContactInfo.Restrictions != nil {
			restrictions = &domain.SharingRestrictions{
				ExpiresAfterHours: req.ContactInfo.Restrictions.ExpiresAfterHours,
				SingleUse:         req.ContactInfo.Restrictions.SingleUse,
				PlatformMediated:  req.ContactInfo.Restrictions.PlatformMediated,
			}
		}

		contactInfo = &domain.ContactInfo{
			Email:           req.ContactInfo.Email,
			Phone:           req.ContactInfo.Phone,
			PreferredMethod: req.ContactInfo.PreferredMethod,
			Message:         req.ContactInfo.Message,
			Restrictions:    restrictions,
		}
	}

	cmd := service.ApproveContactExchangeCommand{
		RequestID:    requestID,
		ApprovalType: domain.ContactExchangeApprovalType(req.ApprovalType),
		ContactInfo:  contactInfo,
	}

	updatedRequest, err := h.contactExchangeService.ApproveContactExchange(c.Request.Context(), cmd)
	if err != nil {
		if domain.IsPostError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve contact exchange request"})
		return
	}

	response := h.toContactExchangeResponseDTO(updatedRequest)
	c.JSON(http.StatusOK, response)
}

// DenyContactExchange denies a contact exchange request
func (h *ContactExchangeHandler) DenyContactExchange(c *gin.Context) {
	requestID, err := domain.ContactExchangeRequestIDFromString(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var req DenyContactExchangeRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Get user ID from context
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userID, err := domain.UserIDFromString(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get the request to verify ownership
	request, err := h.contactExchangeService.GetContactExchangeRequest(c.Request.Context(), requestID)
	if err != nil {
		if domain.IsPostErrorCode(err, domain.ContactExchangeErrorNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contact exchange request not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve contact exchange request"})
		return
	}

	// Verify user is the owner
	if !userID.Equals(request.OwnerUserID()) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the post owner can deny contact exchange requests"})
		return
	}

	cmd := service.DenyContactExchangeCommand{
		RequestID:     requestID,
		DenialReason:  domain.DenialReason(req.DenialReason),
		DenialMessage: req.DenialMessage,
	}

	updatedRequest, err := h.contactExchangeService.DenyContactExchange(c.Request.Context(), cmd)
	if err != nil {
		if domain.IsPostError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deny contact exchange request"})
		return
	}

	response := h.toContactExchangeResponseDTO(updatedRequest)
	c.JSON(http.StatusOK, response)
}

// CancelContactExchange cancels a contact exchange request (for requesters)
func (h *ContactExchangeHandler) CancelContactExchange(c *gin.Context) {
	requestID, err := domain.ContactExchangeRequestIDFromString(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	// Get user ID from context
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userID, err := domain.UserIDFromString(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get the request to verify ownership
	request, err := h.contactExchangeService.GetContactExchangeRequest(c.Request.Context(), requestID)
	if err != nil {
		if domain.IsPostErrorCode(err, domain.ContactExchangeErrorNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contact exchange request not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve contact exchange request"})
		return
	}

	// Verify user is the requester and request is still pending
	if !userID.Equals(request.RequesterUserID()) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the requester can cancel this request"})
		return
	}

	if request.Status() != domain.ContactExchangeStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can only cancel pending requests"})
		return
	}

	// For cancellation, we deny with a special reason
	cmd := service.DenyContactExchangeCommand{
		RequestID:     requestID,
		DenialReason:  domain.DenialReasonUserPreference,
		DenialMessage: stringPtr("Cancelled by requester"),
	}

	updatedRequest, err := h.contactExchangeService.DenyContactExchange(c.Request.Context(), cmd)
	if err != nil {
		if domain.IsPostError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel contact exchange request"})
		return
	}

	response := h.toContactExchangeResponseDTO(updatedRequest)
	c.JSON(http.StatusOK, response)
}

// ListContactExchangeRequests lists contact exchange requests with filtering
func (h *ContactExchangeHandler) ListContactExchangeRequests(c *gin.Context) {
	// Get user ID from context
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userID, err := domain.UserIDFromString(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Parse query parameters
	filters := domain.ContactExchangeFilters{}

	// Role filter: "requester" or "owner"
	role := c.Query("role")
	switch role {
	case "requester":
		filters.RequesterUserID = &userID
	case "owner":
		filters.OwnerUserID = &userID
	default:
		// If no role specified, show both requester and owner requests
		// We'll need to make two queries for this
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role parameter is required (requester or owner)"})
		return
	}

	// Status filter
	if status := c.Query("status"); status != "" {
		contactStatus := domain.ContactExchangeStatus(status)
		filters.Status = &contactStatus
	}

	// Post ID filter
	if postIDStr := c.Query("post_id"); postIDStr != "" {
		postID, err := domain.PostIDFromString(postIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
			return
		}
		filters.PostID = &postID
	}

	// Pagination
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
			return
		}
		filters.Limit = limit
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset parameter"})
			return
		}
		filters.Offset = offset
	}

	requests, err := h.contactExchangeService.ListContactExchangeRequests(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list contact exchange requests"})
		return
	}

	var responses []ContactExchangeResponseDTO
	for _, request := range requests {
		responses = append(responses, h.toContactExchangeResponseDTO(request))
	}

	c.JSON(http.StatusOK, gin.H{
		"requests": responses,
		"pagination": gin.H{
			"limit":  filters.Limit,
			"offset": filters.Offset,
		},
	})
}

func (h *ContactExchangeHandler) toContactExchangeResponseDTO(request *domain.ContactExchangeRequest) ContactExchangeResponseDTO {
	response := ContactExchangeResponseDTO{
		ID:                   request.ID().String(),
		PostID:               request.PostID().String(),
		RequesterUserID:      request.RequesterUserID().String(),
		OwnerUserID:          request.OwnerUserID().String(),
		Status:               string(request.Status()),
		Message:              request.Message(),
		VerificationRequired: request.VerificationRequired(),
		ExpiresAt:            request.ExpiresAt().Format("2006-01-02T15:04:05Z07:00"),
		CreatedAt:            request.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:            request.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}

	if request.VerificationDetails() != nil {
		response.VerificationDetails = &VerificationDetailsDTO{
			Method:       string(request.VerificationDetails().Method),
			Question:     request.VerificationDetails().Question,
			Requirements: request.VerificationDetails().Requirements,
		}
	}

	if request.ApprovalType() != nil {
		approvalType := string(*request.ApprovalType())
		response.ApprovalType = &approvalType
	}

	if request.DenialReason() != nil {
		denialReason := string(*request.DenialReason())
		response.DenialReason = &denialReason
		response.DenialMessage = request.DenialMessage()
	}

	if request.EncryptedContactInfo() != nil {
		contactInfo := &EncryptedContactInfoDTO{
			Email:           request.EncryptedContactInfo().Email,
			Phone:           request.EncryptedContactInfo().Phone,
			PreferredMethod: request.EncryptedContactInfo().PreferredMethod,
			Message:         request.EncryptedContactInfo().Message,
		}

		if request.EncryptedContactInfo().SharingRestrictions != nil {
			contactInfo.SharingRestrictions = &SharingRestrictionsDTO{
				ExpiresAfterHours: request.EncryptedContactInfo().SharingRestrictions.ExpiresAfterHours,
				SingleUse:         request.EncryptedContactInfo().SharingRestrictions.SingleUse,
				PlatformMediated:  request.EncryptedContactInfo().SharingRestrictions.PlatformMediated,
			}
		}

		response.EncryptedContactInfo = contactInfo
	}

	return response
}

func stringPtr(s string) *string {
	return &s
}