package dto

import (
	"database/sql"
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
	req := domain.CreatePhotoRequest{
		PostID:       domain.PostID(dto.PostID),
		URL:          dto.URL,
		Caption:      dto.Caption,
		DisplayOrder: dto.DisplayOrder,
		Format:       dto.Format,
		SizeBytes:    dto.SizeBytes,
	}

	photo, err := domain.NewPhoto(req)
	if err != nil {
		return nil, err
	}

	reconstructedPhoto := domain.ReconstructPhoto(
		domain.PhotoID(dto.ID),
		domain.PostID(dto.PostID),
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
		ID:           string(photo.ID()),
		PostID:       string(photo.PostID()),
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
