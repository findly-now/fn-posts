package dto

import (
	"database/sql"
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

	var organizationID *domain.OrganizationID
	if dto.OrganizationID.Valid {
		orgID := domain.OrganizationID(dto.OrganizationID.String)
		organizationID = &orgID
	}

	post := domain.ReconstructPost(
		domain.PostID(dto.ID),
		dto.Title,
		dto.Description,
		location,
		dto.RadiusMeters,
		domain.PostStatus(dto.Status),
		domain.PostType(dto.Type),
		domain.UserID(dto.CreatedBy),
		organizationID,
		dto.CreatedAt,
		dto.UpdatedAt,
		photos,
	)

	return post, nil
}

func FromDomainPost(post *domain.Post) *PostDTO {
	dto := &PostDTO{
		ID:           string(post.ID()),
		Title:        post.Title(),
		Description:  post.Description(),
		Longitude:    post.Location().Longitude,
		Latitude:     post.Location().Latitude,
		RadiusMeters: post.RadiusMeters(),
		Status:       string(post.Status()),
		Type:         string(post.PostType()),
		CreatedBy:    string(post.CreatedBy()),
		CreatedAt:    post.CreatedAt(),
		UpdatedAt:    post.UpdatedAt(),
	}

	if post.OrganizationID() != nil {
		dto.OrganizationID = sql.NullString{
			String: string(*post.OrganizationID()),
			Valid:  true,
		}
	}

	return dto
}
