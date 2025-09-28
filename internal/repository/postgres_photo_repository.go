package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jsarabia/fn-posts/internal/domain"
)

type PostgresPhotoRepository struct {
	db *sql.DB
}

func NewPostgresPhotoRepository(db *sql.DB) *PostgresPhotoRepository {
	return &PostgresPhotoRepository{db: db}
}

func (r *PostgresPhotoRepository) Save(ctx context.Context, photo *domain.Photo) error {
	query := `
		INSERT INTO post_photos (
			id, post_id, url, thumbnail_url, caption,
			display_order, format, size_bytes, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.ExecContext(
		ctx, query,
		photo.ID, photo.PostID, photo.URL, photo.ThumbnailURL,
		photo.Caption, photo.DisplayOrder, photo.Format,
		photo.SizeBytes, photo.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save photo: %w", err)
	}

	return nil
}

func (r *PostgresPhotoRepository) FindByID(ctx context.Context, id domain.PhotoID) (*domain.Photo, error) {
	query := `
		SELECT id, post_id, url, thumbnail_url, caption,
		       display_order, format, size_bytes, created_at
		FROM post_photos
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)

	var photoID domain.PhotoID
	var postID domain.PostID
	var url, caption, format string
	var displayOrder int
	var sizeBytes int64
	var createdAt time.Time
	var thumbnailURL sql.NullString

	err := row.Scan(
		&photoID, &postID, &url, &thumbnailURL,
		&caption, &displayOrder, &format,
		&sizeBytes, &createdAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("photo not found")
		}
		return nil, fmt.Errorf("failed to find photo: %w", err)
	}

	var thumbnailURLStr string
	if thumbnailURL.Valid {
		thumbnailURLStr = thumbnailURL.String
	}

	photo := domain.ReconstructPhoto(
		photoID, postID, url, thumbnailURLStr, caption,
		displayOrder, format, sizeBytes, createdAt,
	)

	return photo, nil
}

func (r *PostgresPhotoRepository) FindByPostID(ctx context.Context, postID domain.PostID) ([]*domain.Photo, error) {
	query := `
		SELECT id, post_id, url, thumbnail_url, caption,
		       display_order, format, size_bytes, created_at
		FROM post_photos
		WHERE post_id = $1
		ORDER BY display_order`

	rows, err := r.db.QueryContext(ctx, query, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to find photos by post ID: %w", err)
	}
	defer rows.Close()

	var photos []*domain.Photo
	for rows.Next() {
		var photoID domain.PhotoID
		var postID domain.PostID
		var url, caption, format string
		var displayOrder int
		var sizeBytes int64
		var createdAt time.Time
		var thumbnailURL sql.NullString

		err := rows.Scan(
			&photoID, &postID, &url, &thumbnailURL,
			&caption, &displayOrder, &format,
			&sizeBytes, &createdAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan photo: %w", err)
		}

		var thumbnailURLStr string
		if thumbnailURL.Valid {
			thumbnailURLStr = thumbnailURL.String
		}

		photo := domain.ReconstructPhoto(
			photoID, postID, url, thumbnailURLStr, caption,
			displayOrder, format, sizeBytes, createdAt,
		)

		photos = append(photos, photo)
	}

	return photos, nil
}

func (r *PostgresPhotoRepository) Update(ctx context.Context, photo *domain.Photo) error {
	query := `
		UPDATE post_photos SET
			url = $2, thumbnail_url = $3, caption = $4,
			display_order = $5, format = $6, size_bytes = $7
		WHERE id = $1`

	result, err := r.db.ExecContext(
		ctx, query,
		photo.ID, photo.URL, photo.ThumbnailURL, photo.Caption,
		photo.DisplayOrder, photo.Format, photo.SizeBytes,
	)

	if err != nil {
		return fmt.Errorf("failed to update photo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("photo not found")
	}

	return nil
}

func (r *PostgresPhotoRepository) Delete(ctx context.Context, id domain.PhotoID) error {
	query := `DELETE FROM post_photos WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete photo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("photo not found")
	}

	return nil
}
