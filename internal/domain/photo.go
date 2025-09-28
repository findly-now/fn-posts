package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidPhotoURL     = errors.New("invalid photo URL")
	ErrInvalidPhotoFormat  = errors.New("invalid photo format")
	ErrInvalidDisplayOrder = errors.New("invalid display order")
)

type Photo struct {
	ID           PhotoID   `json:"id"`
	PostID       PostID    `json:"post_id"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	Caption      string    `json:"caption,omitempty"`
	DisplayOrder int       `json:"display_order"`
	Format       string    `json:"format"`
	SizeBytes    int64     `json:"size_bytes"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreatePhotoRequest struct {
	PostID       PostID `json:"post_id" binding:"required"`
	URL          string `json:"url" binding:"required"`
	Caption      string `json:"caption" binding:"max=500"`
	DisplayOrder int    `json:"display_order" binding:"min=1,max=10"`
	Format       string `json:"format" binding:"required"`
	SizeBytes    int64  `json:"size_bytes" binding:"min=1"`
}

func NewPhoto(req CreatePhotoRequest) (*Photo, error) {
	if req.URL == "" {
		return nil, ErrInvalidPhotoURL
	}

	if err := validatePhotoFormat(req.Format); err != nil {
		return nil, err
	}

	if req.DisplayOrder < 1 || req.DisplayOrder > 10 {
		return nil, ErrInvalidDisplayOrder
	}

	return &Photo{
		ID:           NewPhotoID(),
		PostID:       req.PostID,
		URL:          req.URL,
		Caption:      req.Caption,
		DisplayOrder: req.DisplayOrder,
		Format:       strings.ToLower(req.Format),
		SizeBytes:    req.SizeBytes,
		CreatedAt:    time.Now(),
	}, nil
}

func (p *Photo) UpdateCaption(caption string) error {
	if len(caption) > 500 {
		return errors.New("caption too long")
	}
	p.Caption = caption
	return nil
}

func (p *Photo) SetThumbnailURL(thumbnailURL string) {
	p.ThumbnailURL = thumbnailURL
}

func validatePhotoFormat(format string) error {
	allowedFormats := map[string]bool{
		"jpg":  true,
		"jpeg": true,
		"png":  true,
		"webp": true,
	}

	if !allowedFormats[strings.ToLower(format)] {
		return ErrInvalidPhotoFormat
	}

	return nil
}

func (p *Photo) IsValidFormat() bool {
	return validatePhotoFormat(p.Format) == nil
}
