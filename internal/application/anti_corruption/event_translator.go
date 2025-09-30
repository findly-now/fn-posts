package anti_corruption

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jsarabia/fn-posts/internal/domain"
)

var (
	ErrInvalidEventType     = errors.New("invalid event type")
	ErrInvalidEventData     = errors.New("invalid event data")
	ErrMissingRequiredField = errors.New("missing required field")
)

type EventTranslator struct {
	emailValidator *regexp.Regexp
	phoneValidator *regexp.Regexp
}

func NewEventTranslator() *EventTranslator {
	return &EventTranslator{
		emailValidator: regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
		phoneValidator: regexp.MustCompile(`^\+?[1-9]\d{1,14}$`),
	}
}

// External event schemas that may come from other services
type ExternalPostCreatedEvent struct {
	EventID   string                `json:"event_id"`
	EventType string                `json:"event_type"`
	Source    string                `json:"source"`
	Timestamp time.Time             `json:"timestamp"`
	Version   int                   `json:"version"`
	Data      ExternalPostEventData `json:"data"`
}

type ExternalPostEventData struct {
	PostID         string                 `json:"post_id"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Location       ExternalLocation       `json:"location"`
	RadiusMeters   int                    `json:"radius_meters"`
	Type           string                 `json:"type"`
	Status         string                 `json:"status"`
	UserID         string                 `json:"user_id"`
	OrganizationID *string                `json:"organization_id,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Photos         []ExternalPhotoData    `json:"photos,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type ExternalLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address,omitempty"`
}

type ExternalPhotoData struct {
	PhotoID      string    `json:"photo_id"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	Caption      string    `json:"caption,omitempty"`
	DisplayOrder int       `json:"display_order"`
	Format       string    `json:"format"`
	SizeBytes    int64     `json:"size_bytes"`
	CreatedAt    time.Time `json:"created_at"`
}

// User notification preferences from external systems
type ExternalUserNotificationEvent struct {
	EventID   string                `json:"event_id"`
	EventType string                `json:"event_type"`
	Source    string                `json:"source"`
	Timestamp time.Time             `json:"timestamp"`
	Data      ExternalUserEventData `json:"data"`
}

type ExternalUserEventData struct {
	UserID      string                 `json:"user_id"`
	Email       string                 `json:"email"`
	Phone       string                 `json:"phone,omitempty"`
	Preferences map[string]interface{} `json:"preferences,omitempty"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type TranslatedPostData struct {
	Title          string
	Description    string
	Location       domain.Location
	RadiusMeters   int
	Type           domain.PostType
	CreatedBy      domain.UserID
	OrganizationID *domain.OrganizationID
}

func (t *EventTranslator) TranslatePostCreatedEvent(event ExternalPostCreatedEvent) (*TranslatedPostData, error) {
	if err := t.validatePostEvent(event); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	location, err := t.translateLocation(event.Data.Location)
	if err != nil {
		return nil, fmt.Errorf("invalid location: %w", err)
	}

	postType, err := t.translatePostType(event.Data.Type)
	if err != nil {
		return nil, fmt.Errorf("invalid post type: %w", err)
	}

	var organizationID *domain.OrganizationID
	if event.Data.OrganizationID != nil && *event.Data.OrganizationID != "" {
		orgID, err := domain.OrganizationIDFromString(*event.Data.OrganizationID)
		if err != nil {
			return nil, fmt.Errorf("invalid organization ID: %w", err)
		}
		organizationID = &orgID
	}

	userID, err := domain.UserIDFromString(event.Data.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	return &TranslatedPostData{
		Title:          t.sanitizeText(event.Data.Title),
		Description:    t.sanitizeText(event.Data.Description),
		Location:       location,
		RadiusMeters:   t.normalizeRadius(event.Data.RadiusMeters),
		Type:           postType,
		CreatedBy:      userID,
		OrganizationID: organizationID,
	}, nil
}

func (t *EventTranslator) TranslatePhotosFromExternal(photos []ExternalPhotoData, postID domain.PostID) ([]domain.Photo, error) {
	var domainPhotos []domain.Photo

	for _, extPhoto := range photos {
		if err := t.validatePhotoData(extPhoto); err != nil {
			return nil, fmt.Errorf("invalid photo data: %w", err)
		}

		photoID, err := domain.PhotoIDFromString(extPhoto.PhotoID)
		if err != nil {
			return nil, fmt.Errorf("invalid photo ID: %w", err)
		}

		photo := domain.ReconstructPhoto(
			photoID,
			postID,
			extPhoto.URL,
			extPhoto.ThumbnailURL,
			t.sanitizeText(extPhoto.Caption),
			extPhoto.DisplayOrder,
			strings.ToLower(extPhoto.Format),
			extPhoto.SizeBytes,
			extPhoto.CreatedAt,
		)

		domainPhotos = append(domainPhotos, *photo)
	}

	return domainPhotos, nil
}

func (t *EventTranslator) validatePostEvent(event ExternalPostCreatedEvent) error {
	if event.EventType == "" {
		return fmt.Errorf("%w: event_type", ErrMissingRequiredField)
	}

	if event.Data.PostID == "" {
		return fmt.Errorf("%w: post_id", ErrMissingRequiredField)
	}

	if event.Data.Title == "" {
		return fmt.Errorf("%w: title", ErrMissingRequiredField)
	}

	if event.Data.UserID == "" {
		return fmt.Errorf("%w: user_id", ErrMissingRequiredField)
	}

	if event.Data.Type == "" {
		return fmt.Errorf("%w: type", ErrMissingRequiredField)
	}

	return nil
}

func (t *EventTranslator) validatePhotoData(photo ExternalPhotoData) error {
	if photo.PhotoID == "" {
		return fmt.Errorf("%w: photo_id", ErrMissingRequiredField)
	}

	if photo.URL == "" {
		return fmt.Errorf("%w: url", ErrMissingRequiredField)
	}

	if photo.Format == "" {
		return fmt.Errorf("%w: format", ErrMissingRequiredField)
	}

	allowedFormats := map[string]bool{
		"jpg": true, "jpeg": true, "png": true, "webp": true,
	}

	if !allowedFormats[strings.ToLower(photo.Format)] {
		return fmt.Errorf("unsupported photo format: %s", photo.Format)
	}

	return nil
}

func (t *EventTranslator) translateLocation(extLoc ExternalLocation) (domain.Location, error) {
	return domain.NewLocation(extLoc.Latitude, extLoc.Longitude)
}

func (t *EventTranslator) translatePostType(typeStr string) (domain.PostType, error) {
	switch strings.ToLower(typeStr) {
	case "lost":
		return domain.PostTypeLost, nil
	case "found":
		return domain.PostTypeFound, nil
	default:
		return "", fmt.Errorf("unknown post type: %s", typeStr)
	}
}

func (t *EventTranslator) sanitizeText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\x00", "")

	if len(text) > 2000 {
		text = text[:2000]
	}

	return text
}

func (t *EventTranslator) normalizeRadius(radius int) int {
	if radius < 100 {
		return 1000
	}
	if radius > 50000 {
		return 50000
	}
	return radius
}

// Translation error types
type TranslationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e TranslationError) Error() string {
	return fmt.Sprintf("translation error for field '%s' with value '%v': %s", e.Field, e.Value, e.Message)
}

func NewTranslationError(field string, value interface{}, message string) *TranslationError {
	return &TranslationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}
