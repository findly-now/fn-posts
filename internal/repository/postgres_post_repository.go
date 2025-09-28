package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jsarabia/fn-posts/internal/domain"
	_ "github.com/lib/pq"
)

type PostgresPostRepository struct {
	db *sql.DB
}

func NewPostgresPostRepository(db *sql.DB) *PostgresPostRepository {
	return &PostgresPostRepository{db: db}
}

func (r *PostgresPostRepository) Save(ctx context.Context, post *domain.Post) error {
	query := `
		INSERT INTO posts (
			id, title, description, location, radius_meters,
			status, type, user_id, organization_id, created_at, updated_at
		) VALUES (
			$1, $2, $3, ST_SetSRID(ST_MakePoint($4, $5), 4326), $6,
			$7, $8, $9, $10, $11, $12
		)`

	_, err := r.db.ExecContext(
		ctx, query,
		post.ID(), post.Title(), post.Description(),
		post.Location().Longitude, post.Location().Latitude, post.RadiusMeters(),
		post.Status(), post.PostType(), post.CreatedBy(), post.OrganizationID(),
		post.CreatedAt(), post.UpdatedAt(),
	)

	if err != nil {
		return domain.ErrRepositoryConnection("save post").WithCause(err)
	}

	// Save photos if any
	for _, photo := range post.Photos() {
		if err := r.savePhoto(ctx, &photo); err != nil {
			return fmt.Errorf("failed to save photo: %w", err)
		}
	}

	return nil
}

func (r *PostgresPostRepository) FindByID(ctx context.Context, id domain.PostID) (*domain.Post, error) {
	query := `
		SELECT
			id, title, description,
			ST_X(location) as longitude,
			ST_Y(location) as latitude,
			radius_meters, status, type, user_id, organization_id,
			created_at, updated_at
		FROM posts
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)

	var postID domain.PostID
	var title, description string
	var longitude, latitude float64
	var radiusMeters int
	var status domain.PostStatus
	var postType domain.PostType
	var createdBy domain.UserID
	var organizationID *domain.OrganizationID
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&postID, &title, &description,
		&longitude, &latitude,
		&radiusMeters, &status, &postType,
		&createdBy, &organizationID,
		&createdAt, &updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrPostNotFound(id)
		}
		return nil, domain.ErrRepositoryConnection("find post").WithCause(err)
	}

	location := domain.Location{
		Latitude:  latitude,
		Longitude: longitude,
	}

	// Load photos
	photos, err := r.findPhotosByPostID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load photos: %w", err)
	}

	post := domain.ReconstructPost(
		postID, title, description, location, radiusMeters,
		status, postType, createdBy, organizationID,
		createdAt, updatedAt, photos,
	)

	return post, nil
}

func (r *PostgresPostRepository) FindByUserID(ctx context.Context, userID domain.UserID, limit, offset int) ([]*domain.Post, error) {
	query := `
		SELECT
			id, title, description,
			ST_X(location) as longitude,
			ST_Y(location) as latitude,
			radius_meters, status, type, user_id, organization_id,
			created_at, updated_at
		FROM posts
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find posts by user: %w", err)
	}
	defer rows.Close()

	return r.scanPosts(ctx, rows)
}

func (r *PostgresPostRepository) FindNearby(ctx context.Context, location domain.Location, radius domain.Distance, postType *domain.PostType, limit, offset int) ([]*domain.Post, error) {
	baseQuery := `
		SELECT
			id, title, description,
			ST_X(location) as longitude,
			ST_Y(location) as latitude,
			radius_meters, status, type, user_id, organization_id,
			created_at, updated_at,
			ST_Distance(location, ST_SetSRID(ST_MakePoint($1, $2), 4326)) as distance
		FROM posts
		WHERE ST_DWithin(
			location,
			ST_SetSRID(ST_MakePoint($1, $2), 4326),
			$3
		)
		AND status = 'active'`

	args := []interface{}{location.Longitude, location.Latitude, radius.Meters}
	argIndex := 4

	if postType != nil {
		baseQuery += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, *postType)
		argIndex++
	}

	baseQuery += fmt.Sprintf(" ORDER BY distance LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby posts: %w", err)
	}
	defer rows.Close()

	return r.scanPostsWithDistance(ctx, rows)
}

func (r *PostgresPostRepository) Update(ctx context.Context, post *domain.Post) error {
	query := `
		UPDATE posts SET
			title = $2, description = $3,
			location = ST_SetSRID(ST_MakePoint($4, $5), 4326),
			radius_meters = $6, status = $7, updated_at = $8
		WHERE id = $1`

	result, err := r.db.ExecContext(
		ctx, query,
		post.ID(), post.Title(), post.Description(),
		post.Location().Longitude, post.Location().Latitude,
		post.RadiusMeters(), post.Status(), post.UpdatedAt(),
	)

	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("post not found")
	}

	return nil
}

func (r *PostgresPostRepository) Delete(ctx context.Context, id domain.PostID) error {
	query := `UPDATE posts SET status = 'deleted', updated_at = NOW() WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("post not found")
	}

	return nil
}

func (r *PostgresPostRepository) List(ctx context.Context, filters domain.PostFilters) ([]*domain.Post, error) {
	filters.SetDefaults()

	query, args := r.buildListQuery(filters)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list posts: %w", err)
	}
	defer rows.Close()

	return r.scanPosts(ctx, rows)
}

func (r *PostgresPostRepository) Count(ctx context.Context, filters domain.PostFilters) (int64, error) {
	query, args := r.buildCountQuery(filters)

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count posts: %w", err)
	}

	return count, nil
}

func (r *PostgresPostRepository) buildListQuery(filters domain.PostFilters) (string, []interface{}) {
	baseQuery := `
		SELECT
			id, title, description,
			ST_X(location) as longitude,
			ST_Y(location) as latitude,
			radius_meters, status, type, user_id, organization_id,
			created_at, updated_at
		FROM posts WHERE 1=1`

	conditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filters.Status)
		argIndex++
	}

	if filters.Type != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, *filters.Type)
		argIndex++
	}

	if filters.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, *filters.UserID)
		argIndex++
	}

	if filters.OrganizationID != nil {
		conditions = append(conditions, fmt.Sprintf("organization_id = $%d", argIndex))
		args = append(args, *filters.OrganizationID)
		argIndex++
	}

	if filters.Location != nil && filters.RadiusMeters != nil {
		conditions = append(conditions, fmt.Sprintf(
			"ST_DWithin(location, ST_SetSRID(ST_MakePoint($%d, $%d), 4326), $%d)",
			argIndex, argIndex+1, argIndex+2))
		args = append(args, filters.Location.Longitude, filters.Location.Latitude, *filters.RadiusMeters)
		argIndex += 3
	}

	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	baseQuery += " ORDER BY created_at DESC"
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, filters.Limit, filters.Offset)

	return baseQuery, args
}

func (r *PostgresPostRepository) buildCountQuery(filters domain.PostFilters) (string, []interface{}) {
	baseQuery := "SELECT COUNT(*) FROM posts WHERE 1=1"

	conditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filters.Status)
		argIndex++
	}

	if filters.Type != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, *filters.Type)
		argIndex++
	}

	if filters.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, *filters.UserID)
		argIndex++
	}

	if filters.OrganizationID != nil {
		conditions = append(conditions, fmt.Sprintf("organization_id = $%d", argIndex))
		args = append(args, *filters.OrganizationID)
		argIndex++
	}

	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	return baseQuery, args
}

func (r *PostgresPostRepository) scanPost(row *sql.Row) (*domain.Post, error) {
	var id domain.PostID
	var title, description string
	var longitude, latitude float64
	var radiusMeters int
	var status domain.PostStatus
	var postType domain.PostType
	var createdBy domain.UserID
	var organizationID *domain.OrganizationID
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&id, &title, &description,
		&longitude, &latitude,
		&radiusMeters, &status, &postType,
		&createdBy, &organizationID,
		&createdAt, &updatedAt,
	)

	if err != nil {
		return nil, err
	}

	location := domain.Location{
		Latitude:  latitude,
		Longitude: longitude,
	}

	post := domain.ReconstructPost(
		id, title, description, location, radiusMeters,
		status, postType, createdBy, organizationID,
		createdAt, updatedAt, []domain.Photo{},
	)

	return post, nil
}

func (r *PostgresPostRepository) scanPosts(ctx context.Context, rows *sql.Rows) ([]*domain.Post, error) {
	var posts []*domain.Post

	for rows.Next() {
		var id domain.PostID
		var title, description string
		var longitude, latitude float64
		var radiusMeters int
		var status domain.PostStatus
		var postType domain.PostType
		var createdBy domain.UserID
		var organizationID *domain.OrganizationID
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&id, &title, &description,
			&longitude, &latitude,
			&radiusMeters, &status, &postType,
			&createdBy, &organizationID,
			&createdAt, &updatedAt,
		)

		if err != nil {
			return nil, err
		}

		location := domain.Location{
			Latitude:  latitude,
			Longitude: longitude,
		}

		// Load photos for each post
		photos, err := r.findPhotosByPostID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to load photos for post %s: %w", id, err)
		}

		post := domain.ReconstructPost(
			id, title, description, location, radiusMeters,
			status, postType, createdBy, organizationID,
			createdAt, updatedAt, photos,
		)

		posts = append(posts, post)
	}

	return posts, nil
}

func (r *PostgresPostRepository) scanPostsWithDistance(ctx context.Context, rows *sql.Rows) ([]*domain.Post, error) {
	var posts []*domain.Post

	for rows.Next() {
		var id domain.PostID
		var title, description string
		var longitude, latitude, distance float64
		var radiusMeters int
		var status domain.PostStatus
		var postType domain.PostType
		var createdBy domain.UserID
		var organizationID *domain.OrganizationID
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&id, &title, &description,
			&longitude, &latitude,
			&radiusMeters, &status, &postType,
			&createdBy, &organizationID,
			&createdAt, &updatedAt,
			&distance,
		)

		if err != nil {
			return nil, err
		}

		location := domain.Location{
			Latitude:  latitude,
			Longitude: longitude,
		}

		// Load photos for each post
		photos, err := r.findPhotosByPostID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to load photos for post %s: %w", id, err)
		}

		post := domain.ReconstructPost(
			id, title, description, location, radiusMeters,
			status, postType, createdBy, organizationID,
			createdAt, updatedAt, photos,
		)

		posts = append(posts, post)
	}

	return posts, nil
}

func (r *PostgresPostRepository) savePhoto(ctx context.Context, photo *domain.Photo) error {
	query := `
		INSERT INTO post_photos (
			id, post_id, url, thumbnail_url, caption,
			display_order, format, size_bytes, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.ExecContext(
		ctx, query,
		photo.ID(), photo.PostID(), photo.URL(), photo.ThumbnailURL(),
		photo.Caption(), photo.DisplayOrder(), photo.Format(),
		photo.SizeBytes(), photo.CreatedAt(),
	)

	return err
}

func (r *PostgresPostRepository) findPhotosByPostID(ctx context.Context, postID domain.PostID) ([]domain.Photo, error) {
	query := `
		SELECT id, post_id, url, thumbnail_url, caption,
		       display_order, format, size_bytes, created_at
		FROM post_photos
		WHERE post_id = $1
		ORDER BY display_order`

	rows, err := r.db.QueryContext(ctx, query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []domain.Photo
	for rows.Next() {
		var photoID domain.PhotoID
		var postID domain.PostID
		var url string
		var thumbnailURL sql.NullString
		var caption string
		var displayOrder int
		var format string
		var sizeBytes int64
		var createdAt time.Time

		err := rows.Scan(
			&photoID, &postID, &url, &thumbnailURL,
			&caption, &displayOrder, &format,
			&sizeBytes, &createdAt,
		)

		if err != nil {
			return nil, err
		}

		thumbnailURLStr := ""
		if thumbnailURL.Valid {
			thumbnailURLStr = thumbnailURL.String
		}

		photo := domain.ReconstructPhoto(
			photoID, postID, url, thumbnailURLStr, caption,
			displayOrder, format, sizeBytes, createdAt,
		)

		photos = append(photos, *photo)
	}

	return photos, nil
}
