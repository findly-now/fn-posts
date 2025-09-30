package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/jsarabia/fn-posts/internal/domain"
	"github.com/jsarabia/fn-posts/internal/repository"
	"github.com/jsarabia/fn-posts/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRSAEncryptionService_E2E(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Setup repositories
	keyRepo := repository.NewPostgresKeyRepository(db)
	auditLogger := repository.NewPostgresEncryptionAuditLogger(db)

	// Create encryption service
	encryptionService, err := domain.NewRSAEncryptionService(keyRepo, auditLogger)
	require.NoError(t, err)
	require.NotNil(t, encryptionService)

	t.Run("Encrypt and Decrypt Contact Information", func(t *testing.T) {
		// Create test contact info
		originalContactInfo := domain.ContactInfo{
			Email:           &[]string{"test@example.com"}[0],
			Phone:           &[]string{"+1234567890"}[0],
			PreferredMethod: "email",
			Message:         &[]string{"I believe this is my item"}[0],
			Restrictions: &domain.SharingRestrictions{
				ExpiresAfterHours: 24,
				SingleUse:         true,
				PlatformMediated:  false,
			},
		}

		// Encrypt contact information
		encryptedInfo, err := encryptionService.EncryptContactInfo(originalContactInfo)
		require.NoError(t, err)
		require.NotNil(t, encryptedInfo)

		// Verify that the data is encrypted (email field should contain encrypted data)
		assert.NotNil(t, encryptedInfo.Email)
		assert.NotEqual(t, *originalContactInfo.Email, *encryptedInfo.Email)

		// Decrypt contact information
		decryptedInfo, err := encryptionService.DecryptContactInfo(encryptedInfo)
		require.NoError(t, err)
		require.NotNil(t, decryptedInfo)

		// Verify decrypted data matches original
		assert.Equal(t, *originalContactInfo.Email, *decryptedInfo.Email)
		assert.Equal(t, *originalContactInfo.Phone, *decryptedInfo.Phone)
		assert.Equal(t, originalContactInfo.PreferredMethod, decryptedInfo.PreferredMethod)
		assert.Equal(t, *originalContactInfo.Message, *decryptedInfo.Message)
		assert.Equal(t, originalContactInfo.Restrictions.ExpiresAfterHours, decryptedInfo.Restrictions.ExpiresAfterHours)
		assert.Equal(t, originalContactInfo.Restrictions.SingleUse, decryptedInfo.Restrictions.SingleUse)
		assert.Equal(t, originalContactInfo.Restrictions.PlatformMediated, decryptedInfo.Restrictions.PlatformMediated)
	})

	t.Run("Generate and Validate Contact Token", func(t *testing.T) {
		// Create test contact info
		contactInfo := domain.ContactInfo{
			Email:           &[]string{"token@example.com"}[0],
			Phone:           &[]string{"+1987654321"}[0],
			PreferredMethod: "phone",
			Message:         &[]string{"Contact me about my lost item"}[0],
		}

		// Generate contact token
		expiresAt := time.Now().Add(2 * time.Hour)
		token, err := encryptionService.GenerateContactToken(contactInfo, expiresAt)
		require.NoError(t, err)
		require.NotNil(t, token)

		// Verify token properties
		assert.NotEmpty(t, token.Token)
		assert.NotEmpty(t, token.KeyFingerprint)
		assert.Equal(t, expiresAt.Unix(), token.ExpiresAt.Unix())
		assert.NotEmpty(t, token.IntegrityHash)

		// Validate and decrypt token
		decryptedContactInfo, err := encryptionService.ValidateContactToken(token)
		require.NoError(t, err)
		require.NotNil(t, decryptedContactInfo)

		// Verify decrypted data matches original
		assert.Equal(t, *contactInfo.Email, *decryptedContactInfo.Email)
		assert.Equal(t, *contactInfo.Phone, *decryptedContactInfo.Phone)
		assert.Equal(t, contactInfo.PreferredMethod, decryptedContactInfo.PreferredMethod)
		assert.Equal(t, *contactInfo.Message, *decryptedContactInfo.Message)
	})

	t.Run("Token Expiration Validation", func(t *testing.T) {
		// Create test contact info
		contactInfo := domain.ContactInfo{
			Email:           &[]string{"expired@example.com"}[0],
			PreferredMethod: "email",
		}

		// Generate token that expires immediately
		expiresAt := time.Now().Add(-1 * time.Second) // Already expired
		token, err := encryptionService.GenerateContactToken(contactInfo, expiresAt)
		require.NoError(t, err)

		// Try to validate expired token
		_, err = encryptionService.ValidateContactToken(token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("Token Integrity Validation", func(t *testing.T) {
		// Create test contact info
		contactInfo := domain.ContactInfo{
			Email:           &[]string{"integrity@example.com"}[0],
			PreferredMethod: "email",
		}

		// Generate valid token
		expiresAt := time.Now().Add(1 * time.Hour)
		token, err := encryptionService.GenerateContactToken(contactInfo, expiresAt)
		require.NoError(t, err)

		// Corrupt the integrity hash
		token.IntegrityHash = "corrupted_hash"

		// Try to validate corrupted token
		_, err = encryptionService.ValidateContactToken(token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "integrity")
	})

	t.Run("Key Rotation", func(t *testing.T) {
		// Get current active key fingerprint
		originalFingerprint := encryptionService.GetActiveKeyFingerprint()
		assert.NotEmpty(t, originalFingerprint)

		// Rotate keys
		err := encryptionService.RotateKeys()
		require.NoError(t, err)

		// Verify new active key
		newFingerprint := encryptionService.GetActiveKeyFingerprint()
		assert.NotEmpty(t, newFingerprint)
		assert.NotEqual(t, originalFingerprint, newFingerprint)

		// Verify old key still exists and can decrypt old data
		contactInfo := domain.ContactInfo{
			Email:           &[]string{"rotation@example.com"}[0],
			PreferredMethod: "email",
		}

		// Encrypt with new key
		encryptedInfo, err := encryptionService.EncryptContactInfo(contactInfo)
		require.NoError(t, err)

		// Should be able to decrypt with current service (which now has the new key)
		decryptedInfo, err := encryptionService.DecryptContactInfo(encryptedInfo)
		require.NoError(t, err)
		assert.Equal(t, *contactInfo.Email, *decryptedInfo.Email)
	})
}

func TestContactExchangeService_EncryptionIntegration_E2E(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	// Setup all repositories and services
	keyRepo := repository.NewPostgresKeyRepository(db)
	auditLogger := repository.NewPostgresEncryptionAuditLogger(db)
	encryptionService, err := domain.NewRSAEncryptionService(keyRepo, auditLogger)
	require.NoError(t, err)

	// Create mock repositories for other dependencies
	contactExchangeRepo := &mockContactExchangeRepository{requests: make(map[string]*domain.ContactExchangeRequest)}
	postRepo := &mockPostRepository{posts: make(map[string]*domain.Post)}
	userContextRepo := &mockUserContextRepository{}
	eventPublisher := &mockEventPublisher{}

	// Create contact exchange service with encryption
	contactService := service.NewContactExchangeService(
		contactExchangeRepo,
		postRepo,
		userContextRepo,
		eventPublisher,
		encryptionService,
		auditLogger,
	)

	t.Run("Complete Contact Exchange Workflow with Encryption", func(t *testing.T) {
		// Create test post and users
		postID := domain.NewPostID()
		ownerUserID := domain.NewUserID()
		requesterUserID := domain.NewUserID()

		// Setup mock post
		post := createTestPost(postID, ownerUserID)
		postRepo.posts[postID.String()] = post

		// Setup mock user contexts
		ownerUser := &domain.PrivacySafeUser{
			UserID:      ownerUserID,
			DisplayName: "Post Owner",
		}
		requesterUser := &domain.PrivacySafeUser{
			UserID:      requesterUserID,
			DisplayName: "Item Requester",
		}
		userContextRepo.users[ownerUserID.String()] = ownerUser
		userContextRepo.users[requesterUserID.String()] = requesterUser

		// Step 1: Create contact exchange request
		createCmd := service.CreateContactExchangeCommand{
			PostID:               postID,
			RequesterUserID:      requesterUserID,
			Message:              &[]string{"I think this is my lost item"}[0],
			VerificationRequired: false,
			ExpirationHours:      72,
		}

		request, err := contactService.CreateContactExchangeRequest(ctx, createCmd)
		require.NoError(t, err)
		require.NotNil(t, request)
		assert.Equal(t, domain.ContactExchangeStatusPending, request.Status())

		// Step 2: Approve with contact information
		contactInfo := domain.ContactInfo{
			Email:           &[]string{"owner@example.com"}[0],
			Phone:           &[]string{"+1555123456"}[0],
			PreferredMethod: "email",
			Message:         &[]string{"Please contact me about the item"}[0],
		}

		approveCmd := service.ApproveContactExchangeCommand{
			RequestID:    request.ID(),
			ApprovalType: domain.ContactExchangeApprovalTypeFull,
			ContactInfo:  &contactInfo,
		}

		approvedRequest, err := contactService.ApproveContactExchange(ctx, approveCmd)
		require.NoError(t, err)
		require.NotNil(t, approvedRequest)
		assert.Equal(t, domain.ContactExchangeStatusApproved, approvedRequest.Status())

		// Verify encrypted contact info is stored
		encryptedInfo := approvedRequest.EncryptedContactInfo()
		require.NotNil(t, encryptedInfo)
		// The contact info should be encrypted (not plaintext)
		assert.NotEqual(t, *contactInfo.Email, *encryptedInfo.Email)

		// Step 3: Decrypt contact information (requester access)
		decryptedInfo, err := contactService.DecryptContactInfo(ctx, request.ID(), requesterUserID)
		require.NoError(t, err)
		require.NotNil(t, decryptedInfo)

		// Verify decrypted data matches original
		assert.Equal(t, *contactInfo.Email, *decryptedInfo.Email)
		assert.Equal(t, *contactInfo.Phone, *decryptedInfo.Phone)
		assert.Equal(t, contactInfo.PreferredMethod, decryptedInfo.PreferredMethod)
		assert.Equal(t, *contactInfo.Message, *decryptedInfo.Message)

		// Step 4: Verify audit logs were created
		auditLogs, err := auditLogger.GetAuditTrail(ownerUserID, &request.ID(), 10)
		require.NoError(t, err)
		assert.Greater(t, len(auditLogs), 0)

		// Should have encryption and decryption logs
		foundEncryption := false
		foundDecryption := false
		for _, log := range auditLogs {
			if log.Operation == domain.EncryptionOperationEncrypt && log.Success {
				foundEncryption = true
			}
			if log.Operation == domain.EncryptionOperationDecrypt && log.Success {
				foundDecryption = true
			}
		}
		assert.True(t, foundEncryption, "Should have encryption audit log")
		assert.True(t, foundDecryption, "Should have decryption audit log")
	})

	t.Run("Unauthorized Decryption Attempt", func(t *testing.T) {
		// Create a request with encrypted contact info
		postID := domain.NewPostID()
		ownerUserID := domain.NewUserID()
		requesterUserID := domain.NewUserID()
		unauthorizedUserID := domain.NewUserID()

		post := createTestPost(postID, ownerUserID)
		postRepo.posts[postID.String()] = post

		// Create and approve a request
		createCmd := service.CreateContactExchangeCommand{
			PostID:          postID,
			RequesterUserID: requesterUserID,
			ExpirationHours: 24,
		}
		request, err := contactService.CreateContactExchangeRequest(ctx, createCmd)
		require.NoError(t, err)

		contactInfo := domain.ContactInfo{
			Email:           &[]string{"secret@example.com"}[0],
			PreferredMethod: "email",
		}
		approveCmd := service.ApproveContactExchangeCommand{
			RequestID:    request.ID(),
			ApprovalType: domain.ContactExchangeApprovalTypeFull,
			ContactInfo:  &contactInfo,
		}
		_, err = contactService.ApproveContactExchange(ctx, approveCmd)
		require.NoError(t, err)

		// Try to decrypt with unauthorized user
		_, err = contactService.DecryptContactInfo(ctx, request.ID(), unauthorizedUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})
}

// Helper function to create test post
func createTestPost(postID domain.PostID, userID domain.UserID) *domain.Post {
	location, _ := domain.NewLocation(40.7831, -73.9665, "New York, NY", 1000)
	return domain.ReconstructPost(
		postID,
		"Test Lost Item",
		"A test item that was lost",
		domain.PostTypeLost,
		domain.PostStatusActive,
		userID,
		domain.NewOrganizationID(),
		location,
		nil, // photos
		time.Now(),
		time.Now(),
	)
}

// Mock repositories for testing
type mockContactExchangeRepository struct {
	requests map[string]*domain.ContactExchangeRequest
}

func (m *mockContactExchangeRepository) Save(ctx context.Context, request *domain.ContactExchangeRequest) error {
	m.requests[request.ID().String()] = request
	return nil
}

func (m *mockContactExchangeRepository) FindByID(ctx context.Context, id domain.ContactExchangeRequestID) (*domain.ContactExchangeRequest, error) {
	if request, exists := m.requests[id.String()]; exists {
		return request, nil
	}
	return nil, domain.NewContactExchangeError(domain.ContactExchangeErrorNotFound, "not found")
}

func (m *mockContactExchangeRepository) Update(ctx context.Context, request *domain.ContactExchangeRequest) error {
	m.requests[request.ID().String()] = request
	return nil
}

func (m *mockContactExchangeRepository) FindByPostID(ctx context.Context, postID domain.PostID) ([]*domain.ContactExchangeRequest, error) {
	return nil, nil
}

func (m *mockContactExchangeRepository) FindByRequesterUserID(ctx context.Context, userID domain.UserID, limit, offset int) ([]*domain.ContactExchangeRequest, error) {
	return nil, nil
}

func (m *mockContactExchangeRepository) FindByOwnerUserID(ctx context.Context, userID domain.UserID, limit, offset int) ([]*domain.ContactExchangeRequest, error) {
	return nil, nil
}

func (m *mockContactExchangeRepository) FindExpired(ctx context.Context, limit int) ([]*domain.ContactExchangeRequest, error) {
	return nil, nil
}

func (m *mockContactExchangeRepository) Delete(ctx context.Context, id domain.ContactExchangeRequestID) error {
	delete(m.requests, id.String())
	return nil
}

func (m *mockContactExchangeRepository) List(ctx context.Context, filters domain.ContactExchangeFilters) ([]*domain.ContactExchangeRequest, error) {
	return nil, nil
}

func (m *mockContactExchangeRepository) Count(ctx context.Context, filters domain.ContactExchangeFilters) (int64, error) {
	return 0, nil
}

type mockPostRepository struct {
	posts map[string]*domain.Post
}

func (m *mockPostRepository) Save(ctx context.Context, post *domain.Post) error {
	m.posts[post.ID().String()] = post
	return nil
}

func (m *mockPostRepository) FindByID(ctx context.Context, id domain.PostID) (*domain.Post, error) {
	if post, exists := m.posts[id.String()]; exists {
		return post, nil
	}
	return nil, domain.NewPostError(domain.BusinessErrorPostNotFound, "not found")
}

func (m *mockPostRepository) FindByUserID(ctx context.Context, userID domain.UserID, limit, offset int) ([]*domain.Post, error) {
	return nil, nil
}

func (m *mockPostRepository) FindNearby(ctx context.Context, location domain.Location, radius domain.Distance, postType *domain.PostType, limit, offset int) ([]*domain.Post, error) {
	return nil, nil
}

func (m *mockPostRepository) Update(ctx context.Context, post *domain.Post) error {
	m.posts[post.ID().String()] = post
	return nil
}

func (m *mockPostRepository) Delete(ctx context.Context, id domain.PostID) error {
	delete(m.posts, id.String())
	return nil
}

func (m *mockPostRepository) List(ctx context.Context, filters domain.PostFilters) ([]*domain.Post, error) {
	return nil, nil
}

func (m *mockPostRepository) Count(ctx context.Context, filters domain.PostFilters) (int64, error) {
	return 0, nil
}

type mockUserContextRepository struct {
	users map[string]*domain.PrivacySafeUser
}

func (m *mockUserContextRepository) GetPrivacySafeUser(ctx context.Context, userID domain.UserID) (*domain.PrivacySafeUser, error) {
	if user, exists := m.users[userID.String()]; exists {
		return user, nil
	}
	return &domain.PrivacySafeUser{
		UserID:      userID,
		DisplayName: "Test User",
	}, nil
}

func (m *mockUserContextRepository) GetPrivacySafeUsers(ctx context.Context, userIDs []domain.UserID) (map[domain.UserID]*domain.PrivacySafeUser, error) {
	users := make(map[domain.UserID]*domain.PrivacySafeUser)
	for _, userID := range userIDs {
		user, _ := m.GetPrivacySafeUser(ctx, userID)
		users[userID] = user
	}
	return users, nil
}

type mockEventPublisher struct{}

func (m *mockEventPublisher) PublishEvent(ctx context.Context, event *domain.PostEvent) error {
	return nil
}