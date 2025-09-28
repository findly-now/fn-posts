package domain

import (
	"errors"
	"strings"
	"time"
)

type Photo struct {
	id           PhotoID
	postID       PostID
	url          string
	thumbnailURL string
	caption      string
	displayOrder int
	format       string
	sizeBytes    int64
	createdAt    time.Time
}

type CreatePhotoRequest struct {
	PostID       PostID
	URL          string
	Caption      string
	DisplayOrder int
	Format       string
	SizeBytes    int64
}

func NewPhoto(req CreatePhotoRequest) (*Photo, error) {
	if req.URL == "" {
		return nil, ErrInvalidPhotoURL(req.URL)
	}

	if err := validatePhotoFormat(req.Format); err != nil {
		return nil, err
	}

	if req.DisplayOrder < 1 || req.DisplayOrder > 10 {
		return nil, ErrInvalidDisplayOrder(req.DisplayOrder)
	}

	return &Photo{
		id:           NewPhotoID(),
		postID:       req.PostID,
		url:          req.URL,
		caption:      req.Caption,
		displayOrder: req.DisplayOrder,
		format:       strings.ToLower(req.Format),
		sizeBytes:    req.SizeBytes,
		createdAt:    time.Now(),
	}, nil
}

func (p *Photo) UpdateCaption(caption string) error {
	if len(caption) > 500 {
		return errors.New("caption too long")
	}
	p.caption = caption
	return nil
}

func (p *Photo) SetThumbnailURL(thumbnailURL string) {
	p.thumbnailURL = thumbnailURL
}

func validatePhotoFormat(format string) error {
	allowedFormats := map[string]bool{
		"jpg":  true,
		"jpeg": true,
		"png":  true,
		"webp": true,
	}

	if !allowedFormats[strings.ToLower(format)] {
		return ErrInvalidPhotoFormat(format)
	}

	return nil
}

func (p *Photo) IsValidFormat() bool {
	return validatePhotoFormat(p.format) == nil
}

func ReconstructPhoto(
	id PhotoID,
	postID PostID,
	url, thumbnailURL, caption string,
	displayOrder int,
	format string,
	sizeBytes int64,
	createdAt time.Time,
) *Photo {
	return &Photo{
		id:           id,
		postID:       postID,
		url:          url,
		thumbnailURL: thumbnailURL,
		caption:      caption,
		displayOrder: displayOrder,
		format:       format,
		sizeBytes:    sizeBytes,
		createdAt:    createdAt,
	}
}

func (p *Photo) ID() PhotoID {
	return p.id
}

func (p *Photo) PostID() PostID {
	return p.postID
}

func (p *Photo) URL() string {
	return p.url
}

func (p *Photo) ThumbnailURL() string {
	return p.thumbnailURL
}

func (p *Photo) Caption() string {
	return p.caption
}

func (p *Photo) DisplayOrder() int {
	return p.displayOrder
}

func (p *Photo) Format() string {
	return p.format
}

func (p *Photo) SizeBytes() int64 {
	return p.sizeBytes
}

func (p *Photo) CreatedAt() time.Time {
	return p.createdAt
}
