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
