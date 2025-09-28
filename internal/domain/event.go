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
	ID        uuid.UUID
	EventType EventType
	PostID    PostID
	UserID    UserID
	TenantID  *OrganizationID
	Data      interface{}
	Timestamp time.Time
	Version   int
}

type PostCreatedEventData struct {
	Post *Post
}

type PostUpdatedEventData struct {
	Post     *Post
	Changes  map[string]interface{}
	Previous map[string]interface{}
}

type PostStatusChangedEventData struct {
	PostID         PostID
	NewStatus      PostStatus
	PreviousStatus PostStatus
}

type PhotoEventData struct {
	PostID PostID
	Photo  *Photo
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
