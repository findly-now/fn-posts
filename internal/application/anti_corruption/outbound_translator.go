package anti_corruption

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jsarabia/fn-posts/internal/domain"
)

type OutboundEventTranslator struct {
	serviceName string
	version     string
}

func NewOutboundEventTranslator(serviceName, version string) *OutboundEventTranslator {
	return &OutboundEventTranslator{
		serviceName: serviceName,
		version:     version,
	}
}

// External event schema for Kafka publishing
type KafkaEvent struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
	Data      interface{}            `json:"data"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// External schemas that will be consumed by fn-matcher, fn-notifications, etc.
type ExternalPostSchema struct {
	PostID         string                 `json:"post_id"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Location       ExternalLocationSchema `json:"location"`
	RadiusMeters   int                    `json:"radius_meters"`
	Type           string                 `json:"type"`
	Status         string                 `json:"status"`
	UserID         string                 `json:"user_id"`
	OrganizationID *string                `json:"organization_id,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Photos         []ExternalPhotoSchema  `json:"photos,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
}

type ExternalLocationSchema struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type ExternalPhotoSchema struct {
	PhotoID      string    `json:"photo_id"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	Caption      string    `json:"caption,omitempty"`
	DisplayOrder int       `json:"display_order"`
	Format       string    `json:"format"`
	SizeBytes    int64     `json:"size_bytes"`
	CreatedAt    time.Time `json:"created_at"`
}

type PostCreatedEventData struct {
	Post ExternalPostSchema `json:"post"`
}

type PostUpdatedEventData struct {
	Post     ExternalPostSchema     `json:"post"`
	Changes  map[string]interface{} `json:"changes"`
	Previous map[string]interface{} `json:"previous"`
}

type PostStatusChangedEventData struct {
	PostID         string    `json:"post_id"`
	NewStatus      string    `json:"new_status"`
	PreviousStatus string    `json:"previous_status"`
	Timestamp      time.Time `json:"timestamp"`
}

type PhotoEventData struct {
	PostID string              `json:"post_id"`
	Photo  ExternalPhotoSchema `json:"photo"`
}

func (t *OutboundEventTranslator) TranslatePostEvent(domainEvent *domain.PostEvent) (*KafkaEvent, error) {
	kafkaEvent := &KafkaEvent{
		EventID:   domainEvent.ID.String(),
		EventType: string(domainEvent.EventType),
		Source:    t.serviceName,
		Timestamp: domainEvent.Timestamp,
		Version:   t.version,
		Metadata: map[string]interface{}{
			"tenant_id": domainEvent.TenantID,
			"user_id":   string(domainEvent.UserID),
		},
	}

	switch domainEvent.EventType {
	case domain.EventTypePostCreated:
		data, ok := domainEvent.Data.(*domain.PostCreatedEventData)
		if !ok {
			return nil, fmt.Errorf("invalid data type for PostCreated event")
		}

		externalPost, err := t.translatePostToExternal(data.Post)
		if err != nil {
			return nil, err
		}

		kafkaEvent.Data = PostCreatedEventData{
			Post: externalPost,
		}

	case domain.EventTypePostUpdated:
		data, ok := domainEvent.Data.(*domain.PostUpdatedEventData)
		if !ok {
			return nil, fmt.Errorf("invalid data type for PostUpdated event")
		}

		externalPost, err := t.translatePostToExternal(data.Post)
		if err != nil {
			return nil, err
		}

		kafkaEvent.Data = PostUpdatedEventData{
			Post:     externalPost,
			Changes:  data.Changes,
			Previous: data.Previous,
		}

	case domain.EventTypePostResolved, domain.EventTypePostDeleted:
		data, ok := domainEvent.Data.(*domain.PostStatusChangedEventData)
		if !ok {
			return nil, fmt.Errorf("invalid data type for PostStatusChanged event")
		}

		kafkaEvent.Data = PostStatusChangedEventData{
			PostID:         string(data.PostID),
			NewStatus:      string(data.NewStatus),
			PreviousStatus: string(data.PreviousStatus),
			Timestamp:      domainEvent.Timestamp,
		}

	case domain.EventTypePhotoAdded, domain.EventTypePhotoRemoved:
		data, ok := domainEvent.Data.(*domain.PhotoEventData)
		if !ok {
			return nil, fmt.Errorf("invalid data type for Photo event")
		}

		externalPhoto := t.translatePhotoToExternal(data.Photo)

		kafkaEvent.Data = PhotoEventData{
			PostID: string(data.PostID),
			Photo:  externalPhoto,
		}

	default:
		return nil, fmt.Errorf("unsupported event type: %s", domainEvent.EventType)
	}

	return kafkaEvent, nil
}

func (t *OutboundEventTranslator) translatePostToExternal(post *domain.Post) (ExternalPostSchema, error) {
	var organizationID *string
	if post.OrganizationID() != nil {
		orgID := string(*post.OrganizationID())
		organizationID = &orgID
	}

	var photos []ExternalPhotoSchema
	for _, photo := range post.Photos() {
		photos = append(photos, t.translatePhotoToExternal(&photo))
	}

	return ExternalPostSchema{
		PostID:      string(post.ID()),
		Title:       post.Title(),
		Description: post.Description(),
		Location: ExternalLocationSchema{
			Latitude:  post.Location().Latitude,
			Longitude: post.Location().Longitude,
		},
		RadiusMeters:   post.RadiusMeters(),
		Type:           string(post.PostType()),
		Status:         string(post.Status()),
		UserID:         string(post.CreatedBy()),
		OrganizationID: organizationID,
		CreatedAt:      post.CreatedAt(),
		UpdatedAt:      post.UpdatedAt(),
		Photos:         photos,
		Tags:           t.extractTagsFromDescription(post.Description()),
	}, nil
}

func (t *OutboundEventTranslator) translatePhotoToExternal(photo *domain.Photo) ExternalPhotoSchema {
	return ExternalPhotoSchema{
		PhotoID:      string(photo.ID()),
		URL:          photo.URL(),
		ThumbnailURL: photo.ThumbnailURL(),
		Caption:      photo.Caption(),
		DisplayOrder: photo.DisplayOrder(),
		Format:       photo.Format(),
		SizeBytes:    photo.SizeBytes(),
		CreatedAt:    photo.CreatedAt(),
	}
}

func (t *OutboundEventTranslator) extractTagsFromDescription(description string) []string {
	// Simple tag extraction - in a real implementation, this might use NLP
	// For now, just extract common lost/found keywords
	tags := []string{}
	lowerDesc := strings.ToLower(description)

	commonTags := map[string]bool{
		"phone":       strings.Contains(lowerDesc, "phone"),
		"wallet":      strings.Contains(lowerDesc, "wallet"),
		"keys":        strings.Contains(lowerDesc, "key"),
		"jewelry":     strings.Contains(lowerDesc, "jewelry") || strings.Contains(lowerDesc, "ring") || strings.Contains(lowerDesc, "necklace"),
		"bag":         strings.Contains(lowerDesc, "bag") || strings.Contains(lowerDesc, "purse"),
		"electronics": strings.Contains(lowerDesc, "laptop") || strings.Contains(lowerDesc, "tablet") || strings.Contains(lowerDesc, "electronic"),
		"clothing":    strings.Contains(lowerDesc, "shirt") || strings.Contains(lowerDesc, "jacket") || strings.Contains(lowerDesc, "clothes"),
		"pet":         strings.Contains(lowerDesc, "dog") || strings.Contains(lowerDesc, "cat") || strings.Contains(lowerDesc, "pet"),
	}

	for tag, found := range commonTags {
		if found {
			tags = append(tags, tag)
		}
	}

	return tags
}

func (t *OutboundEventTranslator) ToJSON(event *KafkaEvent) ([]byte, error) {
	return json.Marshal(event)
}

// Domain Event Publisher that uses the translator
type AntiCorruptionEventPublisher struct {
	translator     *OutboundEventTranslator
	kafkaPublisher KafkaPublisher
}

type KafkaPublisher interface {
	PublishMessage(topic string, key string, message []byte) error
}

func NewAntiCorruptionEventPublisher(translator *OutboundEventTranslator, kafkaPublisher KafkaPublisher) *AntiCorruptionEventPublisher {
	return &AntiCorruptionEventPublisher{
		translator:     translator,
		kafkaPublisher: kafkaPublisher,
	}
}

func (p *AntiCorruptionEventPublisher) PublishEvent(ctx context.Context, domainEvent *domain.PostEvent) error {
	// Translate domain event to external schema
	kafkaEvent, err := p.translator.TranslatePostEvent(domainEvent)
	if err != nil {
		return fmt.Errorf("failed to translate event: %w", err)
	}

	// Serialize to JSON
	eventJSON, err := p.translator.ToJSON(kafkaEvent)
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	// Determine topic based on event type
	topic := p.getTopicForEventType(domainEvent.EventType)

	// Use post ID as partition key for ordering
	key := string(domainEvent.PostID)

	// Publish to Kafka
	if err := p.kafkaPublisher.PublishMessage(topic, key, eventJSON); err != nil {
		return fmt.Errorf("failed to publish to Kafka: %w", err)
	}

	return nil
}

func (p *AntiCorruptionEventPublisher) getTopicForEventType(eventType domain.EventType) string {
	switch eventType {
	case domain.EventTypePostCreated:
		return "posts.created"
	case domain.EventTypePostUpdated:
		return "posts.updated"
	case domain.EventTypePostResolved:
		return "posts.resolved"
	case domain.EventTypePostDeleted:
		return "posts.deleted"
	case domain.EventTypePhotoAdded:
		return "posts.photos.added"
	case domain.EventTypePhotoRemoved:
		return "posts.photos.removed"
	default:
		return "posts.events"
	}
}
