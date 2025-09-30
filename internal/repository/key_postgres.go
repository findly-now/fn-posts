package repository

import (
	"database/sql"
	"fmt"

	"github.com/jsarabia/fn-posts/internal/domain"
)

type PostgresKeyRepository struct {
	db *sql.DB
}

func NewPostgresKeyRepository(db *sql.DB) *PostgresKeyRepository {
	return &PostgresKeyRepository{db: db}
}

func (r *PostgresKeyRepository) SaveKey(key *domain.EncryptionKey) error {
	query := `
		INSERT INTO encryption_keys (
			id, fingerprint, private_key, public_key, is_active, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (fingerprint) DO UPDATE SET
			private_key = EXCLUDED.private_key,
			public_key = EXCLUDED.public_key,
			is_active = EXCLUDED.is_active,
			expires_at = EXCLUDED.expires_at
	`

	_, err := r.db.Exec(
		query,
		key.ID,
		key.Fingerprint,
		key.PrivateKey,
		key.PublicKey,
		key.IsActive,
		key.CreatedAt,
		key.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save encryption key: %w", err)
	}

	return nil
}

func (r *PostgresKeyRepository) GetActiveKey() (*domain.EncryptionKey, error) {
	query := `
		SELECT id, fingerprint, private_key, public_key, is_active, created_at, expires_at
		FROM encryption_keys
		WHERE is_active = true
		ORDER BY created_at DESC
		LIMIT 1
	`

	var key domain.EncryptionKey
	var expiresAt sql.NullTime

	err := r.db.QueryRow(query).Scan(
		&key.ID,
		&key.Fingerprint,
		&key.PrivateKey,
		&key.PublicKey,
		&key.IsActive,
		&key.CreatedAt,
		&expiresAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active encryption key found")
		}
		return nil, fmt.Errorf("failed to get active encryption key: %w", err)
	}

	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}

	return &key, nil
}

func (r *PostgresKeyRepository) GetKeyByFingerprint(fingerprint string) (*domain.EncryptionKey, error) {
	query := `
		SELECT id, fingerprint, private_key, public_key, is_active, created_at, expires_at
		FROM encryption_keys
		WHERE fingerprint = $1
	`

	var key domain.EncryptionKey
	var expiresAt sql.NullTime

	err := r.db.QueryRow(query, fingerprint).Scan(
		&key.ID,
		&key.Fingerprint,
		&key.PrivateKey,
		&key.PublicKey,
		&key.IsActive,
		&key.CreatedAt,
		&expiresAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("encryption key with fingerprint %s not found", fingerprint)
		}
		return nil, fmt.Errorf("failed to get encryption key by fingerprint: %w", err)
	}

	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}

	return &key, nil
}

func (r *PostgresKeyRepository) ListKeys() ([]*domain.EncryptionKey, error) {
	query := `
		SELECT id, fingerprint, private_key, public_key, is_active, created_at, expires_at
		FROM encryption_keys
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list encryption keys: %w", err)
	}
	defer rows.Close()

	var keys []*domain.EncryptionKey

	for rows.Next() {
		var key domain.EncryptionKey
		var expiresAt sql.NullTime

		err := rows.Scan(
			&key.ID,
			&key.Fingerprint,
			&key.PrivateKey,
			&key.PublicKey,
			&key.IsActive,
			&key.CreatedAt,
			&expiresAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan encryption key: %w", err)
		}

		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Time
		}

		keys = append(keys, &key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating encryption keys: %w", err)
	}

	return keys, nil
}

func (r *PostgresKeyRepository) MarkKeyInactive(fingerprint string) error {
	query := `UPDATE encryption_keys SET is_active = false WHERE fingerprint = $1`

	result, err := r.db.Exec(query, fingerprint)
	if err != nil {
		return fmt.Errorf("failed to mark key inactive: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no key found with fingerprint %s", fingerprint)
	}

	return nil
}

func (r *PostgresKeyRepository) SetActiveKey(fingerprint string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Mark all keys as inactive
	_, err = tx.Exec("UPDATE encryption_keys SET is_active = false")
	if err != nil {
		return fmt.Errorf("failed to mark all keys inactive: %w", err)
	}

	// Mark specified key as active
	result, err := tx.Exec("UPDATE encryption_keys SET is_active = true WHERE fingerprint = $1", fingerprint)
	if err != nil {
		return fmt.Errorf("failed to mark key active: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no key found with fingerprint %s", fingerprint)
	}

	return tx.Commit()
}