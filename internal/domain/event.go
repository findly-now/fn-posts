package domain

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventTypePostCreated  EventType = "post.created"
	EventTypePostUpdated  EventType = "post.updated"
	EventTypePostResolved EventType = "post.resolved"
	EventTypePostDeleted  EventType = "post.deleted"
	EventTypePhotoAdded   EventType = "post.photo.added"
	EventTypePhotoRemoved EventType = "post.photo.removed"
)

type PostEvent struct {
	ID        uuid.UUID       `json:"id"`
	EventType EventType       `json:"event_type"`
	PostID    PostID          `json:"post_id"`
	UserID    UserID          `json:"user_id"`
	TenantID  *OrganizationID `json:"tenant_id,omitempty"`
	Data      interface{}     `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
	Version   int             `json:"version"`
}

type PostCreatedEventData struct {
	Post *Post `json:"post"`
}

type PostUpdatedEventData struct {
	Post     *Post                  `json:"post"`
	Changes  map[string]interface{} `json:"changes"`
	Previous map[string]interface{} `json:"previous"`
}

type PostStatusChangedEventData struct {
	PostID         PostID     `json:"post_id"`
	NewStatus      PostStatus `json:"new_status"`
	PreviousStatus PostStatus `json:"previous_status"`
}

type PhotoEventData struct {
	PostID PostID `json:"post_id"`
	Photo  *Photo `json:"photo"`
}

func NewPostEvent(eventType EventType, postID PostID, userID UserID, tenantID *OrganizationID, data interface{}) *PostEvent {
	return &PostEvent{
		ID:        uuid.New(),
		EventType: eventType,
		PostID:    postID,
		UserID:    userID,
		TenantID:  tenantID,
		Data:      data,
		Timestamp: time.Now(),
		Version:   1,
	}
}
