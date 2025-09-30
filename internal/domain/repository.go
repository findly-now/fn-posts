package domain

import (
	"context"
)

type PostRepository interface {
	Save(ctx context.Context, post *Post) error
	FindByID(ctx context.Context, id PostID) (*Post, error)
	FindByUserID(ctx context.Context, userID UserID, limit, offset int) ([]*Post, error)
	FindNearby(ctx context.Context, location Location, radius Distance, postType *PostType, limit, offset int) ([]*Post, error)
	Update(ctx context.Context, post *Post) error
	Delete(ctx context.Context, id PostID) error
	List(ctx context.Context, filters PostFilters) ([]*Post, error)
	Count(ctx context.Context, filters PostFilters) (int64, error)
}

type PhotoRepository interface {
	Save(ctx context.Context, photo *Photo) error
	FindByID(ctx context.Context, id PhotoID) (*Photo, error)
	FindByPostID(ctx context.Context, postID PostID) ([]*Photo, error)
	Update(ctx context.Context, photo *Photo) error
	Delete(ctx context.Context, id PhotoID) error
}

type EventPublisher interface {
	PublishEvent(ctx context.Context, event *PostEvent) error
}

// ContactExchangeRepository manages contact exchange requests
type ContactExchangeRepository interface {
	Save(ctx context.Context, request *ContactExchangeRequest) error
	FindByID(ctx context.Context, id ContactExchangeRequestID) (*ContactExchangeRequest, error)
	FindByPostID(ctx context.Context, postID PostID) ([]*ContactExchangeRequest, error)
	FindByRequesterUserID(ctx context.Context, userID UserID, limit, offset int) ([]*ContactExchangeRequest, error)
	FindByOwnerUserID(ctx context.Context, userID UserID, limit, offset int) ([]*ContactExchangeRequest, error)
	FindExpired(ctx context.Context, limit int) ([]*ContactExchangeRequest, error)
	Update(ctx context.Context, request *ContactExchangeRequest) error
	Delete(ctx context.Context, id ContactExchangeRequestID) error
	List(ctx context.Context, filters ContactExchangeFilters) ([]*ContactExchangeRequest, error)
	Count(ctx context.Context, filters ContactExchangeFilters) (int64, error)
}

// UserContextRepository provides privacy-safe user context for events
type UserContextRepository interface {
	GetPrivacySafeUser(ctx context.Context, userID UserID) (*PrivacySafeUser, error)
	GetPrivacySafeUsers(ctx context.Context, userIDs []UserID) (map[UserID]*PrivacySafeUser, error)
}

// OrganizationContextRepository provides organization context for events
type OrganizationContextRepository interface {
	GetOrganizationData(ctx context.Context, orgID OrganizationID) (*OrganizationData, error)
	GetOrganizationSettings(ctx context.Context, orgID OrganizationID) (*OrganizationSettings, error)
	GetContactSharingPolicy(ctx context.Context, orgID OrganizationID) (*ContactSharingPolicy, error)
}

// KeyRepository manages encryption keys for contact token security
type KeyRepository interface {
	SaveKey(key *EncryptionKey) error
	GetActiveKey() (*EncryptionKey, error)
	GetKeyByFingerprint(fingerprint string) (*EncryptionKey, error)
	ListKeys() ([]*EncryptionKey, error)
	MarkKeyInactive(fingerprint string) error
	SetActiveKey(fingerprint string) error
}

// EncryptionAuditLogger logs encryption operations for compliance
type EncryptionAuditLogger interface {
	LogOperation(log *EncryptionAuditLog) error
	GetAuditTrail(userID UserID, requestID *ContactExchangeRequestID, limit int) ([]*EncryptionAuditLog, error)
}

type PostFilters struct {
	Status         *PostStatus
	Type           *PostType
	UserID         *UserID
	OrganizationID *OrganizationID
	Location       *Location
	RadiusMeters   *int
	CreatedAfter   *string
	CreatedBefore  *string
	Limit          int
	Offset         int
}

func (f *PostFilters) SetDefaults() {
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
}

type ContactExchangeFilters struct {
	Status         *ContactExchangeStatus
	PostID         *PostID
	RequesterUserID *UserID
	OwnerUserID    *UserID
	OrganizationID *OrganizationID
	CreatedAfter   *string
	CreatedBefore  *string
	ExpiresAfter   *string
	ExpiresBefore  *string
	Limit          int
	Offset         int
}

func (f *ContactExchangeFilters) SetDefaults() {
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
}
