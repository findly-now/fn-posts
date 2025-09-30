package service

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jsarabia/fn-posts/internal/domain"
)

type PostService struct {
	postRepo        domain.PostRepository
	photoRepo       domain.PhotoRepository
	userContextRepo domain.UserContextRepository
	orgContextRepo  domain.OrganizationContextRepository
	eventPublisher  domain.EventPublisher
}

// PostServiceConfig holds configuration for enhanced fat event publishing
type PostServiceConfig struct {
	EnableCorrelationIDs bool
	DefaultPrivacyLevel  string
	AIProcessingEnabled  bool
}

func NewPostService(
	postRepo domain.PostRepository,
	photoRepo domain.PhotoRepository,
	userContextRepo domain.UserContextRepository,
	orgContextRepo domain.OrganizationContextRepository,
	eventPublisher domain.EventPublisher,
) *PostService {
	return &PostService{
		postRepo:        postRepo,
		photoRepo:       photoRepo,
		userContextRepo: userContextRepo,
		orgContextRepo:  orgContextRepo,
		eventPublisher:  eventPublisher,
	}
}

func (s *PostService) CreatePost(ctx context.Context, title, description string, photos []domain.Photo, location domain.Location, radiusMeters int, postType domain.PostType, createdBy domain.UserID, organizationID *domain.OrganizationID) (*domain.Post, error) {
	post, err := domain.NewPost(title, description, photos, location, radiusMeters, postType, createdBy, organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid post data: %w", err)
	}

	if err := s.postRepo.Save(ctx, post); err != nil {
		return nil, fmt.Errorf("failed to save post: %w", err)
	}

	// Generate correlation ID for tracing
	correlationID := uuid.New().String()

	// Publish fat PostCreated event with complete context
	if err := s.publishPostCreatedEvent(ctx, post, correlationID); err != nil {
		log.Printf("Failed to publish post created event: %v", err)
		// Don't fail the operation for event publishing errors
	}

	return post, nil
}

// publishPostCreatedEvent creates and publishes a complete fat event for post creation
func (s *PostService) publishPostCreatedEvent(ctx context.Context, post *domain.Post, correlationID string) error {
	// Get privacy-safe user context
	userContext, err := s.userContextRepo.GetPrivacySafeUser(ctx, post.CreatedBy())
	if err != nil {
		log.Printf("Warning: failed to get user context for post creation event: %v", err)
		// Create minimal user context
		userContext = &domain.PrivacySafeUser{
			UserID:      post.CreatedBy(),
			DisplayName: "Unknown User",
			Preferences: domain.UserPreferences{
				Timezone:             "UTC",
				Language:             "en",
				NotificationChannels: []domain.NotificationChannel{},
			},
		}
	}

	// Get organization context if applicable
	var orgContext *domain.OrganizationData
	if post.OrganizationID() != nil {
		orgData, err := s.orgContextRepo.GetOrganizationData(ctx, *post.OrganizationID())
		if err != nil {
			log.Printf("Warning: failed to get organization context for post creation event: %v", err)
		} else {
			orgContext = orgData
		}
	}

	// Create fat event payload
	eventData := &domain.PostCreatedEventData{
		Post:         post.ToPostData(),
		User:         *userContext,
		Organization: orgContext,
		AIAnalysis:   domain.CreateAIMetadataPlaceholder(),
		Triggers:     domain.CreateEventTriggersForPostCreated(),
	}

	// Create event with correlation ID
	event := domain.NewPostEventWithCorrelation(
		domain.EventTypePostCreated,
		post.ID(),
		post.CreatedBy(),
		post.OrganizationID(),
		eventData,
		correlationID,
	)

	// Add privacy context
	event.Privacy = domain.CreatePrivacyContext(nil, "organization_members")

	return s.eventPublisher.PublishEvent(ctx, event)
}

func (s *PostService) GetPostByID(ctx context.Context, id domain.PostID) (*domain.Post, error) {
	post, err := s.postRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find post: %w", err)
	}

	return post, nil
}

func (s *PostService) GetPostsByUser(ctx context.Context, userID domain.UserID, limit, offset int) ([]*domain.Post, error) {
	posts, err := s.postRepo.FindByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find posts by user: %w", err)
	}

	return posts, nil
}

func (s *PostService) SearchNearbyPosts(ctx context.Context, location domain.Location, radiusMeters int, postType *domain.PostType, limit, offset int) ([]*domain.Post, error) {
	if radiusMeters <= 0 {
		radiusMeters = 1000
	}

	if radiusMeters > 50000 {
		radiusMeters = 50000
	}

	radius := domain.Distance{Meters: float64(radiusMeters)}

	posts, err := s.postRepo.FindNearby(ctx, location, radius, postType, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search nearby posts: %w", err)
	}

	return posts, nil
}

func (s *PostService) UpdatePost(ctx context.Context, id domain.PostID, title, description string) (*domain.Post, error) {
	post, err := s.postRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find post: %w", err)
	}

	previousData := map[string]interface{}{
		"title":       post.Title(),
		"description": post.Description(),
	}

	if err := post.Update(title, description); err != nil {
		return nil, fmt.Errorf("failed to update post: %w", err)
	}

	if err := s.postRepo.Update(ctx, post); err != nil {
		return nil, fmt.Errorf("failed to save updated post: %w", err)
	}

	// Publish event
	changes := map[string]interface{}{
		"title":       post.Title(),
		"description": post.Description(),
	}

	event := domain.NewPostEvent(
		domain.EventTypePostUpdated,
		post.ID(),
		post.CreatedBy(),
		post.OrganizationID(),
		&domain.PostUpdatedEventData{
			Post:     post.ToPostData(),
			User:     domain.ToPrivacySafeUser(post.CreatedBy(), "User Name", domain.UserPreferences{
				Timezone: "UTC",
				Language: "en",
				NotificationChannels: []domain.NotificationChannel{domain.NotificationChannelEmail},
			}, nil),
			Changes:  changes,
			Previous: previousData,
		},
	)

	if err := s.eventPublisher.PublishEvent(ctx, event); err != nil {
		log.Printf("Failed to publish post updated event: %v", err)
	}

	return post, nil
}

func (s *PostService) UpdatePostStatus(ctx context.Context, id domain.PostID, newStatus domain.PostStatus) (*domain.Post, error) {
	post, err := s.postRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find post: %w", err)
	}

	previousStatus := post.Status()

	if err := post.UpdateStatus(newStatus); err != nil {
		return nil, fmt.Errorf("failed to update post status: %w", err)
	}

	if err := s.postRepo.Update(ctx, post); err != nil {
		return nil, fmt.Errorf("failed to save updated post: %w", err)
	}

	var eventType domain.EventType
	switch newStatus {
	case domain.PostStatusResolved:
		eventType = domain.EventTypePostResolved
	case domain.PostStatusDeleted:
		eventType = domain.EventTypePostDeleted
	default:
		eventType = domain.EventTypePostUpdated
	}

	event := domain.NewPostEvent(
		eventType,
		post.ID(),
		post.CreatedBy(),
		post.OrganizationID(),
		&domain.PostStatusChangedEventData{
			Post:           post.ToPostData(),
			User:           domain.ToPrivacySafeUser(post.CreatedBy(), "User Name", domain.UserPreferences{
				Timezone: "UTC",
				Language: "en",
				NotificationChannels: []domain.NotificationChannel{domain.NotificationChannelEmail},
			}, nil),
			NewStatus:      newStatus,
			PreviousStatus: previousStatus,
		},
	)

	if err := s.eventPublisher.PublishEvent(ctx, event); err != nil {
		log.Printf("Failed to publish post status changed event: %v", err)
	}

	return post, nil
}

func (s *PostService) DeletePost(ctx context.Context, id domain.PostID) error {
	post, err := s.postRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find post: %w", err)
	}

	if err := s.postRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	event := domain.NewPostEvent(
		domain.EventTypePostDeleted,
		post.ID(),
		post.CreatedBy(),
		post.OrganizationID(),
		&domain.PostStatusChangedEventData{
			Post:           post.ToPostData(),
			User:           domain.ToPrivacySafeUser(post.CreatedBy(), "User Name", domain.UserPreferences{
				Timezone: "UTC",
				Language: "en",
				NotificationChannels: []domain.NotificationChannel{domain.NotificationChannelEmail},
			}, nil),
			NewStatus:      domain.PostStatusDeleted,
			PreviousStatus: post.Status(),
		},
	)

	if err := s.eventPublisher.PublishEvent(ctx, event); err != nil {
		log.Printf("Failed to publish post deleted event: %v", err)
	}

	return nil
}

func (s *PostService) AddPhotoToPost(ctx context.Context, postID domain.PostID, photoReq domain.CreatePhotoRequest) (*domain.Photo, error) {
	post, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to find post: %w", err)
	}

	if len(post.Photos()) >= 10 {
		return nil, domain.ErrInvalidPhotoCount(len(post.Photos()))
	}

	photoReq.PostID = postID
	photoReq.DisplayOrder = len(post.Photos()) + 1

	photo, err := domain.NewPhoto(photoReq)
	if err != nil {
		return nil, fmt.Errorf("invalid photo data: %w", err)
	}

	if err := s.photoRepo.Save(ctx, photo); err != nil {
		return nil, fmt.Errorf("failed to save photo: %w", err)
	}

	if err := post.AddPhoto(*photo); err != nil {
		return nil, fmt.Errorf("failed to add photo to post: %w", err)
	}

	if err := s.postRepo.Update(ctx, post); err != nil {
		return nil, fmt.Errorf("failed to update post: %w", err)
	}

	// Get privacy-safe user context for fat events
	_, err = s.userContextRepo.GetPrivacySafeUser(ctx, post.CreatedBy())
	if err != nil {
		// Log warning but don't fail - continue with event without user context
		log.Printf("Warning: failed to get user context for photo added event: %v", err)
	}

	// Publish fat PhotoAdded event with complete context
	event := domain.NewPostEvent(
		domain.EventTypePhotoAdded,
		post.ID(),
		post.CreatedBy(),
		post.OrganizationID(),
		&domain.PhotoAddedEventData{
			Post:                post.ToPostData(),
			Photo:               photo.ToPhotoData(),
			User:                domain.ToPrivacySafeUser(post.CreatedBy(), "User Name", domain.UserPreferences{
				Timezone: "UTC",
				Language: "en",
				NotificationChannels: []domain.NotificationChannel{domain.NotificationChannelEmail},
			}, nil),
			AIProcessingTrigger: true, // Default to trigger AI processing
		},
	)

	if err := s.eventPublisher.PublishEvent(ctx, event); err != nil {
		log.Printf("Failed to publish photo added event: %v", err)
	}

	return photo, nil
}

func (s *PostService) RemovePhotoFromPost(ctx context.Context, postID domain.PostID, photoID domain.PhotoID) error {
	post, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return fmt.Errorf("failed to find post: %w", err)
	}

	photo, err := s.photoRepo.FindByID(ctx, photoID)
	if err != nil {
		return fmt.Errorf("failed to find photo: %w", err)
	}

	if !photo.PostID().Equals(postID) {
		return fmt.Errorf("photo does not belong to this post")
	}

	if len(post.Photos()) <= 1 {
		return fmt.Errorf("cannot remove last photo from post")
	}

	if err := s.photoRepo.Delete(ctx, photoID); err != nil {
		return fmt.Errorf("failed to delete photo: %w", err)
	}

	if err := post.RemovePhoto(photoID); err != nil {
		return fmt.Errorf("failed to remove photo from post: %w", err)
	}

	if err := s.postRepo.Update(ctx, post); err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	event := domain.NewPostEvent(
		domain.EventTypePhotoRemoved,
		post.ID(),
		post.CreatedBy(),
		post.OrganizationID(),
		&domain.PhotoRemovedEventData{
			Post:  post.ToPostData(),
			Photo: photo.ToPhotoData(),
			User:  domain.ToPrivacySafeUser(post.CreatedBy(), "User Name", domain.UserPreferences{
				Timezone: "UTC",
				Language: "en",
				NotificationChannels: []domain.NotificationChannel{domain.NotificationChannelEmail},
			}, nil),
		},
	)

	if err := s.eventPublisher.PublishEvent(ctx, event); err != nil {
		log.Printf("Failed to publish photo removed event: %v", err)
	}

	return nil
}

func (s *PostService) ListPosts(ctx context.Context, filters domain.PostFilters) ([]*domain.Post, error) {
	filters.SetDefaults()

	posts, err := s.postRepo.List(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list posts: %w", err)
	}

	return posts, nil
}

func (s *PostService) CountPosts(ctx context.Context, filters domain.PostFilters) (int64, error) {
	count, err := s.postRepo.Count(ctx, filters)
	if err != nil {
		return 0, fmt.Errorf("failed to count posts: %w", err)
	}

	return count, nil
}
