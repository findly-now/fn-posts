package domain

import (
	"errors"
	"time"
)

type PostType string
type PostStatus string

const (
	PostTypeLost  PostType = "lost"
	PostTypeFound PostType = "found"
)

const (
	PostStatusActive   PostStatus = "active"
	PostStatusResolved PostStatus = "resolved"
	PostStatusExpired  PostStatus = "expired"
	PostStatusDeleted  PostStatus = "deleted"
)

type Post struct {
	id             PostID
	title          string
	description    string
	photos         []Photo
	location       Location
	radiusMeters   int
	status         PostStatus
	postType       PostType
	createdBy      UserID
	organizationID *OrganizationID
	createdAt      time.Time
	updatedAt      time.Time
}

func NewPost(
	title, description string,
	photos []Photo,
	location Location,
	radiusMeters int,
	postType PostType,
	createdBy UserID,
	organizationID *OrganizationID,
) (*Post, error) {
	if err := validatePostType(postType); err != nil {
		return nil, err
	}

	if title == "" {
		return nil, ErrInvalidTitle()
	}

	if len(photos) < 1 || len(photos) > 10 {
		return nil, ErrInvalidPhotoCount(len(photos))
	}

	if err := location.Validate(); err != nil {
		return nil, err
	}

	if radiusMeters < 100 || radiusMeters > 50000 {
		radiusMeters = 1000
	}

	now := time.Now()

	return &Post{
		id:             NewPostID(),
		title:          title,
		description:    description,
		photos:         photos,
		location:       location,
		radiusMeters:   radiusMeters,
		status:         PostStatusActive,
		postType:       postType,
		createdBy:      createdBy,
		organizationID: organizationID,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

func ReconstructPost(
	id PostID,
	title, description string,
	location Location,
	radiusMeters int,
	status PostStatus,
	postType PostType,
	createdBy UserID,
	organizationID *OrganizationID,
	createdAt, updatedAt time.Time,
	photos []Photo,
) *Post {
	return &Post{
		id:             id,
		title:          title,
		description:    description,
		photos:         photos,
		location:       location,
		radiusMeters:   radiusMeters,
		status:         status,
		postType:       postType,
		createdBy:      createdBy,
		organizationID: organizationID,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}
}

func (p *Post) AddPhoto(photo Photo) error {
	if len(p.photos) >= 10 {
		return ErrInvalidPhotoCount(len(p.photos))
	}

	p.photos = append(p.photos, photo)
	p.updatedAt = time.Now()
	return nil
}

func (p *Post) RemovePhoto(photoID PhotoID) error {
	for i, photo := range p.photos {
		if photo.ID().Equals(photoID) {
			p.photos = append(p.photos[:i], p.photos[i+1:]...)
			p.updatedAt = time.Now()
			return nil
		}
	}
	return errors.New("photo not found")
}

func (p *Post) UpdateStatus(newStatus PostStatus) error {
	if err := p.validateStatusTransition(newStatus); err != nil {
		return err
	}

	p.status = newStatus
	p.updatedAt = time.Now()
	return nil
}

func (p *Post) Update(title, description string) error {
	if title == "" {
		return ErrInvalidTitle()
	}

	p.title = title
	p.description = description
	p.updatedAt = time.Now()
	return nil
}

func (p *Post) IsExpired(expiryDuration time.Duration) bool {
	return time.Since(p.createdAt) > expiryDuration
}

func (p *Post) validateStatusTransition(newStatus PostStatus) error {
	validTransitions := map[PostStatus][]PostStatus{
		PostStatusActive:   {PostStatusResolved, PostStatusExpired, PostStatusDeleted},
		PostStatusResolved: {PostStatusActive, PostStatusDeleted},
		PostStatusExpired:  {PostStatusActive, PostStatusDeleted},
		PostStatusDeleted:  {},
	}

	allowedStatuses, exists := validTransitions[p.status]
	if !exists {
		return ErrCannotTransitionStatus(p.status, newStatus)
	}

	for _, allowed := range allowedStatuses {
		if allowed == newStatus {
			return nil
		}
	}

	return ErrCannotTransitionStatus(p.status, newStatus)
}

func validatePostType(postType PostType) error {
	switch postType {
	case PostTypeLost, PostTypeFound:
		return nil
	default:
		return ErrInvalidPostType(string(postType))
	}
}

func (ps PostStatus) IsValid() bool {
	switch ps {
	case PostStatusActive, PostStatusResolved, PostStatusExpired, PostStatusDeleted:
		return true
	default:
		return false
	}
}

func (p *Post) ID() PostID {
	return p.id
}

func (p *Post) Title() string {
	return p.title
}

func (p *Post) Description() string {
	return p.description
}

func (p *Post) Photos() []Photo {
	return p.photos
}

func (p *Post) Location() Location {
	return p.location
}

func (p *Post) RadiusMeters() int {
	return p.radiusMeters
}

func (p *Post) Status() PostStatus {
	return p.status
}

func (p *Post) PostType() PostType {
	return p.postType
}

func (p *Post) CreatedBy() UserID {
	return p.createdBy
}

func (p *Post) OrganizationID() *OrganizationID {
	return p.organizationID
}

func (p *Post) CreatedAt() time.Time {
	return p.createdAt
}

func (p *Post) UpdatedAt() time.Time {
	return p.updatedAt
}
