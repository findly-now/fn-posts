package dto

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jsarabia/fn-posts/internal/domain"
)

type PhotoDTO struct {
	ID           string         `json:"id" db:"id"`
	PostID       string         `json:"post_id" db:"post_id"`
	URL          string         `json:"url" db:"url"`
	ThumbnailURL sql.NullString `json:"thumbnail_url,omitempty" db:"thumbnail_url"`
	Caption      string         `json:"caption,omitempty" db:"caption"`
	DisplayOrder int            `json:"display_order" db:"display_order"`
	Format       string         `json:"format" db:"format"`
	SizeBytes    int64          `json:"size_bytes" db:"size_bytes"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
}

func (dto *PhotoDTO) ToDomain() (*domain.Photo, error) {
	postID, err := domain.PostIDFromString(dto.PostID)
	if err != nil {
		return nil, fmt.Errorf("invalid post ID: %w", err)
	}

	req := domain.CreatePhotoRequest{
		PostID:       postID,
		URL:          dto.URL,
		Caption:      dto.Caption,
		DisplayOrder: dto.DisplayOrder,
		Format:       dto.Format,
		SizeBytes:    dto.SizeBytes,
	}

	_, err = domain.NewPhoto(req)
	if err != nil {
		return nil, err
	}

	photoID, err := domain.PhotoIDFromString(dto.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid photo ID: %w", err)
	}

	reconstructedPhoto := domain.ReconstructPhoto(
		photoID,
		postID,
		dto.URL,
		dto.ThumbnailURL.String,
		dto.Caption,
		dto.DisplayOrder,
		dto.Format,
		dto.SizeBytes,
		dto.CreatedAt,
	)

	return reconstructedPhoto, nil
}

func FromDomainPhoto(photo *domain.Photo) *PhotoDTO {
	dto := &PhotoDTO{
		ID:           photo.ID().String(),
		PostID:       photo.PostID().String(),
		URL:          photo.URL(),
		Caption:      photo.Caption(),
		DisplayOrder: photo.DisplayOrder(),
		Format:       photo.Format(),
		SizeBytes:    photo.SizeBytes(),
		CreatedAt:    photo.CreatedAt(),
	}

	if thumbnailURL := photo.ThumbnailURL(); thumbnailURL != "" {
		dto.ThumbnailURL = sql.NullString{
			String: thumbnailURL,
			Valid:  true,
		}
	}

	return dto
}
