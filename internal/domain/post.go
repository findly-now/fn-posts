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

var (
	ErrInvalidPostType        = errors.New("invalid post type")
	ErrInvalidPostStatus      = errors.New("invalid post status")
	ErrInvalidPhotoCount      = errors.New("post must have between 1 and 10 photos")
	ErrInvalidTitle           = errors.New("title cannot be empty")
	ErrInvalidLocation        = errors.New("invalid location coordinates")
	ErrCannotTransitionStatus = errors.New("invalid status transition")
)

type Post struct {
	ID             PostID          `json:"id"`
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Photos         []Photo         `json:"photos"`
	Location       Location        `json:"location"`
	RadiusMeters   int             `json:"radius_meters"`
	Status         PostStatus      `json:"status"`
	Type           PostType        `json:"type"`
	CreatedBy      UserID          `json:"created_by"`
	OrganizationID *OrganizationID `json:"organization_id,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type CreatePostRequest struct {
	Title          string          `json:"title" binding:"required,min=1,max=200"`
	Description    string          `json:"description" binding:"max=2000"`
	Location       Location        `json:"location" binding:"required"`
	RadiusMeters   int             `json:"radius_meters" binding:"min=100,max=50000"`
	Type           PostType        `json:"type" binding:"required"`
	CreatedBy      UserID          `json:"created_by" binding:"required"`
	OrganizationID *OrganizationID `json:"organization_id,omitempty"`
}

func NewPost(req CreatePostRequest) (*Post, error) {
	if err := validatePostType(req.Type); err != nil {
		return nil, err
	}

	if req.Title == "" {
		return nil, ErrInvalidTitle
	}

	if err := req.Location.Validate(); err != nil {
		return nil, err
	}

	if req.RadiusMeters < 100 || req.RadiusMeters > 50000 {
		req.RadiusMeters = 1000 // Default to 1km
	}

	now := time.Now()

	return &Post{
		ID:             NewPostID(),
		Title:          req.Title,
		Description:    req.Description,
		Photos:         []Photo{},
		Location:       req.Location,
		RadiusMeters:   req.RadiusMeters,
		Status:         PostStatusActive,
		Type:           req.Type,
		CreatedBy:      req.CreatedBy,
		OrganizationID: req.OrganizationID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (p *Post) AddPhoto(photo Photo) error {
	if len(p.Photos) >= 10 {
		return ErrInvalidPhotoCount
	}

	p.Photos = append(p.Photos, photo)
	p.UpdatedAt = time.Now()
	return nil
}

func (p *Post) RemovePhoto(photoID PhotoID) error {
	for i, photo := range p.Photos {
		if photo.ID.Equals(photoID) {
			p.Photos = append(p.Photos[:i], p.Photos[i+1:]...)
			p.UpdatedAt = time.Now()
			return nil
		}
	}
	return errors.New("photo not found")
}

func (p *Post) UpdateStatus(newStatus PostStatus) error {
	if err := p.validateStatusTransition(newStatus); err != nil {
		return err
	}

	p.Status = newStatus
	p.UpdatedAt = time.Now()
	return nil
}

func (p *Post) Update(title, description string) error {
	if title == "" {
		return ErrInvalidTitle
	}

	p.Title = title
	p.Description = description
	p.UpdatedAt = time.Now()
	return nil
}

func (p *Post) IsExpired(expiryDuration time.Duration) bool {
	return time.Since(p.CreatedAt) > expiryDuration
}

func (p *Post) validateStatusTransition(newStatus PostStatus) error {
	validTransitions := map[PostStatus][]PostStatus{
		PostStatusActive:   {PostStatusResolved, PostStatusExpired, PostStatusDeleted},
		PostStatusResolved: {PostStatusActive, PostStatusDeleted},
		PostStatusExpired:  {PostStatusActive, PostStatusDeleted},
		PostStatusDeleted:  {}, // No transitions from deleted
	}

	allowedStatuses, exists := validTransitions[p.Status]
	if !exists {
		return ErrCannotTransitionStatus
	}

	for _, allowed := range allowedStatuses {
		if allowed == newStatus {
			return nil
		}
	}

	return ErrCannotTransitionStatus
}

func validatePostType(postType PostType) error {
	switch postType {
	case PostTypeLost, PostTypeFound:
		return nil
	default:
		return ErrInvalidPostType
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
