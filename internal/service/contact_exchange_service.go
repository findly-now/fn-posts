package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jsarabia/fn-posts/internal/domain"
)

type ContactExchangeService struct {
	contactExchangeRepo domain.ContactExchangeRepository
	postRepo            domain.PostRepository
	userContextRepo     domain.UserContextRepository
	eventPublisher      domain.EventPublisher
	encryptionService   domain.EncryptionService
	auditLogger         domain.EncryptionAuditLogger
}

func NewContactExchangeService(
	contactExchangeRepo domain.ContactExchangeRepository,
	postRepo domain.PostRepository,
	userContextRepo domain.UserContextRepository,
	eventPublisher domain.EventPublisher,
	encryptionService domain.EncryptionService,
	auditLogger domain.EncryptionAuditLogger,
) *ContactExchangeService {
	return &ContactExchangeService{
		contactExchangeRepo: contactExchangeRepo,
		postRepo:            postRepo,
		userContextRepo:     userContextRepo,
		eventPublisher:      eventPublisher,
		encryptionService:   encryptionService,
		auditLogger:         auditLogger,
	}
}

type CreateContactExchangeCommand struct {
	PostID               domain.PostID
	RequesterUserID      domain.UserID
	Message              *string
	VerificationRequired bool
	VerificationDetails  *domain.VerificationDetails
	ExpirationHours      int
}

type ApproveContactExchangeCommand struct {
	RequestID    domain.ContactExchangeRequestID
	ApprovalType domain.ContactExchangeApprovalType
	ContactInfo  *domain.ContactInfo
}

type DenyContactExchangeCommand struct {
	RequestID     domain.ContactExchangeRequestID
	DenialReason  domain.DenialReason
	DenialMessage *string
}

func (s *ContactExchangeService) CreateContactExchangeRequest(ctx context.Context, cmd CreateContactExchangeCommand) (*domain.ContactExchangeRequest, error) {
	// Validate post exists and get owner
	post, err := s.postRepo.FindByID(ctx, cmd.PostID)
	if err != nil {
		return nil, fmt.Errorf("failed to find post: %w", err)
	}

	if post.Status() != domain.PostStatusActive {
		return nil, domain.NewPostError(domain.BusinessErrorPostNotFound, "Post is not active")
	}

	// Create contact exchange request
	request, err := domain.NewContactExchangeRequest(
		cmd.PostID,
		cmd.RequesterUserID,
		post.CreatedBy(),
		cmd.Message,
		cmd.VerificationRequired,
		cmd.VerificationDetails,
		cmd.ExpirationHours,
	)
	if err != nil {
		return nil, err
	}

	// Save request
	if err := s.contactExchangeRepo.Save(ctx, request); err != nil {
		return nil, fmt.Errorf("failed to save contact exchange request: %w", err)
	}

	// Get privacy-safe user contexts for event
	requester, err := s.userContextRepo.GetPrivacySafeUser(ctx, cmd.RequesterUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get requester user context: %w", err)
	}

	owner, err := s.userContextRepo.GetPrivacySafeUser(ctx, post.CreatedBy())
	if err != nil {
		return nil, fmt.Errorf("failed to get owner user context: %w", err)
	}

	// Publish ContactExchangeRequested event
	eventData := &domain.ContactExchangeRequestedEventData{
		ContactRequest: request.ToContactRequestData(),
		RelatedPost:    post.ToPostData(),
		Requester:      domain.ToPrivacySafeUserExtendedFromUser(requester),
		Owner:          domain.ToPrivacySafeUserExtendedFromUser(owner),
		NotificationRequirements: &domain.NotificationRequirements{
			ImmediateNotification: true,
		},
	}

	event := domain.NewContactExchangeEvent(
		domain.EventTypeContactExchangeRequested,
		request.ID(),
		cmd.RequesterUserID,
		post.OrganizationID(),
		eventData,
	)

	if err := s.eventPublisher.PublishEvent(ctx, event); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to publish ContactExchangeRequested event: %v\n", err)
	}

	return request, nil
}

func (s *ContactExchangeService) ApproveContactExchange(ctx context.Context, cmd ApproveContactExchangeCommand) (*domain.ContactExchangeRequest, error) {
	// Find request
	request, err := s.contactExchangeRepo.FindByID(ctx, cmd.RequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to find contact exchange request: %w", err)
	}

	// Encrypt contact information using RSA-4096
	encryptedContactInfo, err := s.encryptionService.EncryptContactInfo(*cmd.ContactInfo)
	if err != nil {
		// Log encryption failure
		requestID := request.ID()
		errorMessage := err.Error()
		s.auditLogger.LogOperation(&domain.EncryptionAuditLog{
			Operation:      domain.EncryptionOperationEncrypt,
			UserID:         request.OwnerUserID(),
			RequestID:      &requestID,
			KeyFingerprint: s.encryptionService.GetActiveKeyFingerprint(),
			Success:        false,
			ErrorMessage:   &errorMessage,
		})
		return nil, fmt.Errorf("failed to encrypt contact information: %w", err)
	}

	// Log successful encryption
	requestID := request.ID()
	s.auditLogger.LogOperation(&domain.EncryptionAuditLog{
		Operation:      domain.EncryptionOperationEncrypt,
		UserID:         request.OwnerUserID(),
		RequestID:      &requestID,
		KeyFingerprint: s.encryptionService.GetActiveKeyFingerprint(),
		Success:        true,
	})

	// Approve request with encrypted contact info
	if err := request.Approve(cmd.ApprovalType, encryptedContactInfo); err != nil {
		return nil, err
	}

	// Update request
	if err := s.contactExchangeRepo.Update(ctx, request); err != nil {
		return nil, fmt.Errorf("failed to update contact exchange request: %w", err)
	}

	// Get related post and user contexts for event
	post, err := s.postRepo.FindByID(ctx, request.PostID())
	if err != nil {
		return nil, fmt.Errorf("failed to find post: %w", err)
	}

	requester, err := s.userContextRepo.GetPrivacySafeUser(ctx, request.RequesterUserID())
	if err != nil {
		return nil, fmt.Errorf("failed to get requester user context: %w", err)
	}

	owner, err := s.userContextRepo.GetPrivacySafeUser(ctx, request.OwnerUserID())
	if err != nil {
		return nil, fmt.Errorf("failed to get owner user context: %w", err)
	}

	// Publish ContactExchangeApproved event
	contactApproval := &domain.ContactApproval{
		RequestID:             request.ID(),
		ApprovalType:          cmd.ApprovalType,
		ContactInfo:           encryptedContactInfo,
		ExpiresAt:             request.ExpiresAt(),
		ApprovedAt:            time.Now(),
		VerificationCompleted: !request.VerificationRequired(),
	}

	eventData := &domain.ContactExchangeApprovedEventData{
		ContactApproval: contactApproval.ToContactApprovalData(),
		RelatedPost:     post.ToPostData(),
		Requester:       domain.ToPrivacySafeUserExtendedFromUser(requester),
		Owner:           domain.ToPrivacySafeUserExtendedFromUser(owner),
		NotificationRequirements: &domain.NotificationRequirements{
			ImmediateNotification: true,
		},
		AuditTrail: &domain.AuditTrail{
			ApprovalSource: "manual",
		},
	}

	event := domain.NewContactExchangeEvent(
		domain.EventTypeContactExchangeApproved,
		request.ID(),
		request.OwnerUserID(),
		post.OrganizationID(),
		eventData,
	)

	if err := s.eventPublisher.PublishEvent(ctx, event); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to publish ContactExchangeApproved event: %v\n", err)
	}

	return request, nil
}

func (s *ContactExchangeService) DenyContactExchange(ctx context.Context, cmd DenyContactExchangeCommand) (*domain.ContactExchangeRequest, error) {
	// Find request
	request, err := s.contactExchangeRepo.FindByID(ctx, cmd.RequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to find contact exchange request: %w", err)
	}

	// Deny request
	if err := request.Deny(cmd.DenialReason, cmd.DenialMessage); err != nil {
		return nil, err
	}

	// Update request
	if err := s.contactExchangeRepo.Update(ctx, request); err != nil {
		return nil, fmt.Errorf("failed to update contact exchange request: %w", err)
	}

	// Get related post and user contexts for event
	post, err := s.postRepo.FindByID(ctx, request.PostID())
	if err != nil {
		return nil, fmt.Errorf("failed to find post: %w", err)
	}

	requester, err := s.userContextRepo.GetPrivacySafeUser(ctx, request.RequesterUserID())
	if err != nil {
		return nil, fmt.Errorf("failed to get requester user context: %w", err)
	}

	owner, err := s.userContextRepo.GetPrivacySafeUser(ctx, request.OwnerUserID())
	if err != nil {
		return nil, fmt.Errorf("failed to get owner user context: %w", err)
	}

	// Publish ContactExchangeDenied event
	contactDenial := &domain.ContactDenial{
		RequestID:     request.ID(),
		DenialReason:  cmd.DenialReason,
		DenialMessage: cmd.DenialMessage,
		DeniedAt:      time.Now(),
		DenialSource:  "manual",
	}

	eventData := &domain.ContactExchangeDeniedEventData{
		ContactDenial: contactDenial.ToContactDenialData(),
		RelatedPost:   post.ToPostData(),
		Requester:     domain.ToPrivacySafeUserExtendedFromUser(requester),
		Owner:         domain.ToPrivacySafeUserExtendedFromUser(owner),
		NotificationRequirements: &domain.NotificationRequirements{
			ImmediateNotification: true,
		},
	}

	event := domain.NewContactExchangeEvent(
		domain.EventTypeContactExchangeDenied,
		request.ID(),
		request.OwnerUserID(),
		post.OrganizationID(),
		eventData,
	)

	if err := s.eventPublisher.PublishEvent(ctx, event); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to publish ContactExchangeDenied event: %v\n", err)
	}

	return request, nil
}

func (s *ContactExchangeService) GetContactExchangeRequest(ctx context.Context, requestID domain.ContactExchangeRequestID) (*domain.ContactExchangeRequest, error) {
	return s.contactExchangeRepo.FindByID(ctx, requestID)
}

func (s *ContactExchangeService) ListContactExchangeRequests(ctx context.Context, filters domain.ContactExchangeFilters) ([]*domain.ContactExchangeRequest, error) {
	return s.contactExchangeRepo.List(ctx, filters)
}

// DecryptContactInfo decrypts the contact information from an approved contact exchange request
func (s *ContactExchangeService) DecryptContactInfo(ctx context.Context, requestID domain.ContactExchangeRequestID, userID domain.UserID) (*domain.ContactInfo, error) {
	// Find request
	request, err := s.contactExchangeRepo.FindByID(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to find contact exchange request: %w", err)
	}

	// Verify authorization - only requester or owner can decrypt
	if !request.RequesterUserID().Equals(userID) && !request.OwnerUserID().Equals(userID) {
		return nil, fmt.Errorf("unauthorized to decrypt contact information")
	}

	// Verify request is approved
	if request.Status() != domain.ContactExchangeStatusApproved {
		return nil, fmt.Errorf("contact exchange request not approved")
	}

	// Check if request has expired
	if request.IsExpired() {
		return nil, fmt.Errorf("contact exchange request has expired")
	}

	// Get encrypted contact info
	encryptedInfo := request.EncryptedContactInfo()
	if encryptedInfo == nil {
		return nil, fmt.Errorf("no contact information available")
	}

	// Decrypt using encryption service
	contactInfo, err := s.encryptionService.DecryptContactInfo(encryptedInfo)
	if err != nil {
		// Log decryption failure
		errorMessage := err.Error()
		s.auditLogger.LogOperation(&domain.EncryptionAuditLog{
			Operation:      domain.EncryptionOperationDecrypt,
			UserID:         userID,
			RequestID:      &requestID,
			KeyFingerprint: s.encryptionService.GetActiveKeyFingerprint(),
			Success:        false,
			ErrorMessage:   &errorMessage,
		})
		return nil, fmt.Errorf("failed to decrypt contact information: %w", err)
	}

	// Log successful decryption
	s.auditLogger.LogOperation(&domain.EncryptionAuditLog{
		Operation:      domain.EncryptionOperationDecrypt,
		UserID:         userID,
		RequestID:      &requestID,
		KeyFingerprint: s.encryptionService.GetActiveKeyFingerprint(),
		Success:        true,
	})

	return contactInfo, nil
}

// GenerateContactToken creates a secure token for contact exchange
func (s *ContactExchangeService) GenerateContactToken(ctx context.Context, contactInfo domain.ContactInfo, expiresAt time.Time, userID domain.UserID) (*domain.ContactToken, error) {
	token, err := s.encryptionService.GenerateContactToken(contactInfo, expiresAt)
	if err != nil {
		// Log token creation failure
		errorMessage := err.Error()
		s.auditLogger.LogOperation(&domain.EncryptionAuditLog{
			Operation:      domain.EncryptionOperationTokenCreate,
			UserID:         userID,
			KeyFingerprint: s.encryptionService.GetActiveKeyFingerprint(),
			Success:        false,
			ErrorMessage:   &errorMessage,
		})
		return nil, fmt.Errorf("failed to generate contact token: %w", err)
	}

	// Log successful token creation
	s.auditLogger.LogOperation(&domain.EncryptionAuditLog{
		Operation:      domain.EncryptionOperationTokenCreate,
		UserID:         userID,
		KeyFingerprint: s.encryptionService.GetActiveKeyFingerprint(),
		Success:        true,
	})

	return token, nil
}

// ValidateContactToken validates and decrypts a contact token
func (s *ContactExchangeService) ValidateContactToken(ctx context.Context, token *domain.ContactToken, userID domain.UserID) (*domain.ContactInfo, error) {
	contactInfo, err := s.encryptionService.ValidateContactToken(token)
	if err != nil {
		// Log token validation failure
		errorMessage := err.Error()
		s.auditLogger.LogOperation(&domain.EncryptionAuditLog{
			Operation:      domain.EncryptionOperationTokenValidate,
			UserID:         userID,
			KeyFingerprint: token.KeyFingerprint,
			Success:        false,
			ErrorMessage:   &errorMessage,
		})
		return nil, fmt.Errorf("failed to validate contact token: %w", err)
	}

	// Log successful token validation
	s.auditLogger.LogOperation(&domain.EncryptionAuditLog{
		Operation:      domain.EncryptionOperationTokenValidate,
		UserID:         userID,
		KeyFingerprint: token.KeyFingerprint,
		Success:        true,
	})

	return contactInfo, nil
}

func (s *ContactExchangeService) ProcessExpiredRequests(ctx context.Context) error {
	// Find expired requests
	expiredRequests, err := s.contactExchangeRepo.FindExpired(ctx, 100)
	if err != nil {
		return fmt.Errorf("failed to find expired requests: %w", err)
	}

	for _, request := range expiredRequests {
		if err := s.expireContactExchangeRequest(ctx, request); err != nil {
			// Log error but continue processing other requests
			fmt.Printf("Warning: failed to expire contact exchange request %s: %v\n", request.ID().String(), err)
		}
	}

	return nil
}

// CleanupExpiredTokens securely removes encrypted contact data from expired requests
func (s *ContactExchangeService) CleanupExpiredTokens(ctx context.Context) error {
	// Find expired requests that still have encrypted contact info
	expiredRequests, err := s.contactExchangeRepo.FindExpired(ctx, 100)
	if err != nil {
		return fmt.Errorf("failed to find expired requests for cleanup: %w", err)
	}

	cleanedCount := 0
	for _, request := range expiredRequests {
		if request.EncryptedContactInfo() != nil && request.Status() == domain.ContactExchangeStatusExpired {
			// Securely clear the encrypted contact information
			if err := s.securelyCleanupContactInfo(ctx, request); err != nil {
				fmt.Printf("Warning: failed to cleanup contact info for request %s: %v\n", request.ID().String(), err)
				continue
			}
			cleanedCount++
		}
	}

	fmt.Printf("Cleaned up encrypted contact information for %d expired requests\n", cleanedCount)
	return nil
}

// securelyCleanupContactInfo removes encrypted contact data and logs the operation
func (s *ContactExchangeService) securelyCleanupContactInfo(ctx context.Context, request *domain.ContactExchangeRequest) error {
	// Clear the encrypted contact information using domain method
	if err := request.ClearContactInfo(); err != nil {
		// Log cleanup failure
		requestID := request.ID()
		errorMessage := err.Error()
		s.auditLogger.LogOperation(&domain.EncryptionAuditLog{
			Operation:      domain.EncryptionOperationDecrypt, // Using decrypt since we're accessing encrypted data
			UserID:         request.OwnerUserID(),
			RequestID:      &requestID,
			KeyFingerprint: "cleanup_operation",
			Success:        false,
			ErrorMessage:   &errorMessage,
		})
		return fmt.Errorf("failed to clear contact info: %w", err)
	}

	// Update the request in the database
	if err := s.contactExchangeRepo.Update(ctx, request); err != nil {
		return fmt.Errorf("failed to update request after clearing contact info: %w", err)
	}

	// Log successful cleanup operation for audit
	requestID := request.ID()
	s.auditLogger.LogOperation(&domain.EncryptionAuditLog{
		Operation:      domain.EncryptionOperationDecrypt, // Using decrypt since we're accessing encrypted data
		UserID:         request.OwnerUserID(),
		RequestID:      &requestID,
		KeyFingerprint: "cleanup_operation",
		Success:        true,
	})

	return nil
}

func (s *ContactExchangeService) expireContactExchangeRequest(ctx context.Context, request *domain.ContactExchangeRequest) error {
	// Mark as expired
	if err := request.Expire(); err != nil {
		return err
	}

	// Update request
	if err := s.contactExchangeRepo.Update(ctx, request); err != nil {
		return fmt.Errorf("failed to update expired request: %w", err)
	}

	// Get related post and user contexts for event
	post, err := s.postRepo.FindByID(ctx, request.PostID())
	if err != nil {
		return fmt.Errorf("failed to find post: %w", err)
	}

	requester, err := s.userContextRepo.GetPrivacySafeUser(ctx, request.RequesterUserID())
	if err != nil {
		return fmt.Errorf("failed to get requester user context: %w", err)
	}

	owner, err := s.userContextRepo.GetPrivacySafeUser(ctx, request.OwnerUserID())
	if err != nil {
		return fmt.Errorf("failed to get owner user context: %w", err)
	}

	// Calculate duration
	duration := time.Since(request.CreatedAt())

	// Publish ContactExchangeExpired event
	contactExpiration := &domain.ContactExpiration{
		RequestID:        request.ID(),
		OriginalStatus:   domain.ContactExchangeStatusPending, // Assuming it was pending
		ExpirationReason: "timeout",
		ExpiredAt:        time.Now(),
		DurationHours:    duration.Hours(),
	}

	eventData := &domain.ContactExchangeExpiredEventData{
		ContactExpiration: contactExpiration.ToContactExpirationData(),
		RelatedPost:       post.ToPostData(),
		InvolvedUsers: domain.InvolvedUsersExtended{
			Requester: domain.ToPrivacySafeUserExtendedFromUser(requester),
			Owner:     domain.ToPrivacySafeUserExtendedFromUser(owner),
		},
		CleanupActions: &domain.CleanupActions{
			RevokeContactAccess: true,
			ArchiveRequestData:  true,
			UpdateAnalytics:     true,
		},
	}

	event := domain.NewContactExchangeEvent(
		domain.EventTypeContactExchangeExpired,
		request.ID(),
		request.OwnerUserID(),
		post.OrganizationID(),
		eventData,
	)

	if err := s.eventPublisher.PublishEvent(ctx, event); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to publish ContactExchangeExpired event: %v\n", err)
	}

	return nil
}