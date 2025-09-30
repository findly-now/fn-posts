package dto

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jsarabia/fn-posts/internal/domain"
)

type PostDTO struct {
	ID             string          `json:"id" db:"id"`
	Title          string          `json:"title" db:"title"`
	Description    string          `json:"description" db:"description"`
	Longitude      float64         `json:"longitude" db:"longitude"`
	Latitude       float64         `json:"latitude" db:"latitude"`
	RadiusMeters   int             `json:"radius_meters" db:"radius_meters"`
	Status         string          `json:"status" db:"status"`
	Type           string          `json:"type" db:"type"`
	CreatedBy      string          `json:"created_by" db:"user_id"`
	OrganizationID sql.NullString  `json:"organization_id,omitempty" db:"organization_id"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
	Distance       sql.NullFloat64 `json:"distance,omitempty" db:"distance"`
}

func (dto *PostDTO) ToDomain(photos []domain.Photo) (*domain.Post, error) {
	location, err := domain.NewLocation(dto.Latitude, dto.Longitude)
	if err != nil {
		return nil, err
	}

	postID, err := domain.PostIDFromString(dto.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid post ID: %w", err)
	}

	userID, err := domain.UserIDFromString(dto.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	var organizationID *domain.OrganizationID
	if dto.OrganizationID.Valid {
		orgID, err := domain.OrganizationIDFromString(dto.OrganizationID.String)
		if err != nil {
			return nil, fmt.Errorf("invalid organization ID: %w", err)
		}
		organizationID = &orgID
	}

	post := domain.ReconstructPost(
		postID,
		dto.Title,
		dto.Description,
		location,
		dto.RadiusMeters,
		domain.PostStatus(dto.Status),
		domain.PostType(dto.Type),
		userID,
		organizationID,
		dto.CreatedAt,
		dto.UpdatedAt,
		photos,
	)

	return post, nil
}

func FromDomainPost(post *domain.Post) *PostDTO {
	dto := &PostDTO{
		ID:           post.ID().String(),
		Title:        post.Title(),
		Description:  post.Description(),
		Longitude:    post.Location().Longitude,
		Latitude:     post.Location().Latitude,
		RadiusMeters: post.RadiusMeters(),
		Status:       string(post.Status()),
		Type:         string(post.PostType()),
		CreatedBy:    post.CreatedBy().String(),
		CreatedAt:    post.CreatedAt(),
		UpdatedAt:    post.UpdatedAt(),
	}

	if post.OrganizationID() != nil {
		dto.OrganizationID = sql.NullString{
			String: post.OrganizationID().String(),
			Valid:  true,
		}
	}

	return dto
}
