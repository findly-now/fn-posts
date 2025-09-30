package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jsarabia/fn-posts/internal/domain"
)

type PostgresContactExchangeRepository struct {
	db *sql.DB
}

func NewPostgresContactExchangeRepository(db *sql.DB) *PostgresContactExchangeRepository {
	return &PostgresContactExchangeRepository{db: db}
}

func (r *PostgresContactExchangeRepository) Save(ctx context.Context, request *domain.ContactExchangeRequest) error {
	query := `
		INSERT INTO contact_exchange_requests (
			id, post_id, requester_user_id, owner_user_id, status, message,
			verification_required, verification_method, verification_question, verification_requirements,
			approval_type, denial_reason, denial_message, encrypted_contact_info,
			expires_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)`

	var verificationMethod *string
	var verificationQuestion *string
	var verificationRequirements []byte
	var approvalType *string
	var denialReason *string
	var encryptedContactInfo []byte

	if request.VerificationDetails() != nil {
		verificationMethodStr := string(request.VerificationDetails().Method)
		verificationMethod = &verificationMethodStr
		verificationQuestion = request.VerificationDetails().Question

		if len(request.VerificationDetails().Requirements) > 0 {
			var err error
			verificationRequirements, err = json.Marshal(request.VerificationDetails().Requirements)
			if err != nil {
				return fmt.Errorf("failed to marshal verification requirements: %w", err)
			}
		}
	}

	if request.ApprovalType() != nil {
		approvalTypeStr := string(*request.ApprovalType())
		approvalType = &approvalTypeStr
	}

	if request.DenialReason() != nil {
		denialReasonStr := string(*request.DenialReason())
		denialReason = &denialReasonStr
	}

	if request.EncryptedContactInfo() != nil {
		var err error
		encryptedContactInfo, err = json.Marshal(request.EncryptedContactInfo())
		if err != nil {
			return fmt.Errorf("failed to marshal encrypted contact info: %w", err)
		}
	}

	_, err := r.db.ExecContext(ctx, query,
		request.ID().UUID(),
		request.PostID().UUID(),
		request.RequesterUserID().UUID(),
		request.OwnerUserID().UUID(),
		string(request.Status()),
		request.Message(),
		request.VerificationRequired(),
		verificationMethod,
		verificationQuestion,
		verificationRequirements,
		approvalType,
		denialReason,
		request.DenialMessage(),
		encryptedContactInfo,
		request.ExpiresAt(),
		request.CreatedAt(),
		request.UpdatedAt(),
	)

	if err != nil {
		return fmt.Errorf("failed to save contact exchange request: %w", err)
	}

	return nil
}

func (r *PostgresContactExchangeRepository) FindByID(ctx context.Context, id domain.ContactExchangeRequestID) (*domain.ContactExchangeRequest, error) {
	query := `
		SELECT id, post_id, requester_user_id, owner_user_id, status, message,
			   verification_required, verification_method, verification_question, verification_requirements,
			   approval_type, denial_reason, denial_message, encrypted_contact_info,
			   expires_at, created_at, updated_at
		FROM contact_exchange_requests
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id.UUID())
	return r.scanContactExchangeRequest(row)
}

func (r *PostgresContactExchangeRepository) FindByPostID(ctx context.Context, postID domain.PostID) ([]*domain.ContactExchangeRequest, error) {
	query := `
		SELECT id, post_id, requester_user_id, owner_user_id, status, message,
			   verification_required, verification_method, verification_question, verification_requirements,
			   approval_type, denial_reason, denial_message, encrypted_contact_info,
			   expires_at, created_at, updated_at
		FROM contact_exchange_requests
		WHERE post_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, postID.UUID())
	if err != nil {
		return nil, fmt.Errorf("failed to find contact exchange requests by post ID: %w", err)
	}
	defer rows.Close()

	return r.scanContactExchangeRequests(rows)
}

func (r *PostgresContactExchangeRepository) FindByRequesterUserID(ctx context.Context, userID domain.UserID, limit, offset int) ([]*domain.ContactExchangeRequest, error) {
	query := `
		SELECT id, post_id, requester_user_id, owner_user_id, status, message,
			   verification_required, verification_method, verification_question, verification_requirements,
			   approval_type, denial_reason, denial_message, encrypted_contact_info,
			   expires_at, created_at, updated_at
		FROM contact_exchange_requests
		WHERE requester_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID.UUID(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find contact exchange requests by requester user ID: %w", err)
	}
	defer rows.Close()

	return r.scanContactExchangeRequests(rows)
}

func (r *PostgresContactExchangeRepository) FindByOwnerUserID(ctx context.Context, userID domain.UserID, limit, offset int) ([]*domain.ContactExchangeRequest, error) {
	query := `
		SELECT id, post_id, requester_user_id, owner_user_id, status, message,
			   verification_required, verification_method, verification_question, verification_requirements,
			   approval_type, denial_reason, denial_message, encrypted_contact_info,
			   expires_at, created_at, updated_at
		FROM contact_exchange_requests
		WHERE owner_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID.UUID(), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find contact exchange requests by owner user ID: %w", err)
	}
	defer rows.Close()

	return r.scanContactExchangeRequests(rows)
}

func (r *PostgresContactExchangeRepository) FindExpired(ctx context.Context, limit int) ([]*domain.ContactExchangeRequest, error) {
	query := `
		SELECT id, post_id, requester_user_id, owner_user_id, status, message,
			   verification_required, verification_method, verification_question, verification_requirements,
			   approval_type, denial_reason, denial_message, encrypted_contact_info,
			   expires_at, created_at, updated_at
		FROM contact_exchange_requests
		WHERE expires_at <= NOW()
		  AND status IN ('pending', 'approved')
		ORDER BY expires_at ASC
		LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find expired contact exchange requests: %w", err)
	}
	defer rows.Close()

	return r.scanContactExchangeRequests(rows)
}

func (r *PostgresContactExchangeRepository) Update(ctx context.Context, request *domain.ContactExchangeRequest) error {
	query := `
		UPDATE contact_exchange_requests SET
			status = $2,
			message = $3,
			verification_required = $4,
			verification_method = $5,
			verification_question = $6,
			verification_requirements = $7,
			approval_type = $8,
			denial_reason = $9,
			denial_message = $10,
			encrypted_contact_info = $11,
			expires_at = $12,
			updated_at = $13
		WHERE id = $1`

	var verificationMethod *string
	var verificationQuestion *string
	var verificationRequirements []byte
	var approvalType *string
	var denialReason *string
	var encryptedContactInfo []byte

	if request.VerificationDetails() != nil {
		verificationMethodStr := string(request.VerificationDetails().Method)
		verificationMethod = &verificationMethodStr
		verificationQuestion = request.VerificationDetails().Question

		if len(request.VerificationDetails().Requirements) > 0 {
			var err error
			verificationRequirements, err = json.Marshal(request.VerificationDetails().Requirements)
			if err != nil {
				return fmt.Errorf("failed to marshal verification requirements: %w", err)
			}
		}
	}

	if request.ApprovalType() != nil {
		approvalTypeStr := string(*request.ApprovalType())
		approvalType = &approvalTypeStr
	}

	if request.DenialReason() != nil {
		denialReasonStr := string(*request.DenialReason())
		denialReason = &denialReasonStr
	}

	if request.EncryptedContactInfo() != nil {
		var err error
		encryptedContactInfo, err = json.Marshal(request.EncryptedContactInfo())
		if err != nil {
			return fmt.Errorf("failed to marshal encrypted contact info: %w", err)
		}
	}

	_, err := r.db.ExecContext(ctx, query,
		request.ID().UUID(),
		string(request.Status()),
		request.Message(),
		request.VerificationRequired(),
		verificationMethod,
		verificationQuestion,
		verificationRequirements,
		approvalType,
		denialReason,
		request.DenialMessage(),
		encryptedContactInfo,
		request.ExpiresAt(),
		request.UpdatedAt(),
	)

	if err != nil {
		return fmt.Errorf("failed to update contact exchange request: %w", err)
	}

	return nil
}

func (r *PostgresContactExchangeRepository) Delete(ctx context.Context, id domain.ContactExchangeRequestID) error {
	query := `DELETE FROM contact_exchange_requests WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id.UUID())
	if err != nil {
		return fmt.Errorf("failed to delete contact exchange request: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrContactExchangeNotFound(id)
	}

	return nil
}

func (r *PostgresContactExchangeRepository) List(ctx context.Context, filters domain.ContactExchangeFilters) ([]*domain.ContactExchangeRequest, error) {
	filters.SetDefaults()

	query := `
		SELECT id, post_id, requester_user_id, owner_user_id, status, message,
			   verification_required, verification_method, verification_question, verification_requirements,
			   approval_type, denial_reason, denial_message, encrypted_contact_info,
			   expires_at, created_at, updated_at
		FROM contact_exchange_requests`

	whereClause, args := r.buildWhereClause(filters)
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	query += " ORDER BY created_at DESC LIMIT $" + fmt.Sprintf("%d", len(args)+1) + " OFFSET $" + fmt.Sprintf("%d", len(args)+2)
	args = append(args, filters.Limit, filters.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list contact exchange requests: %w", err)
	}
	defer rows.Close()

	return r.scanContactExchangeRequests(rows)
}

func (r *PostgresContactExchangeRepository) Count(ctx context.Context, filters domain.ContactExchangeFilters) (int64, error) {
	query := "SELECT COUNT(*) FROM contact_exchange_requests"

	whereClause, args := r.buildWhereClause(filters)
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count contact exchange requests: %w", err)
	}

	return count, nil
}

func (r *PostgresContactExchangeRepository) buildWhereClause(filters domain.ContactExchangeFilters) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argCount := 0

	if filters.Status != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("status = $%d", argCount))
		args = append(args, string(*filters.Status))
	}

	if filters.PostID != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("post_id = $%d", argCount))
		args = append(args, filters.PostID.UUID())
	}

	if filters.RequesterUserID != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("requester_user_id = $%d", argCount))
		args = append(args, filters.RequesterUserID.UUID())
	}

	if filters.OwnerUserID != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("owner_user_id = $%d", argCount))
		args = append(args, filters.OwnerUserID.UUID())
	}

	if filters.CreatedAfter != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argCount))
		args = append(args, *filters.CreatedAfter)
	}

	if filters.CreatedBefore != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argCount))
		args = append(args, *filters.CreatedBefore)
	}

	if filters.ExpiresAfter != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("expires_at >= $%d", argCount))
		args = append(args, *filters.ExpiresAfter)
	}

	if filters.ExpiresBefore != nil {
		argCount++
		conditions = append(conditions, fmt.Sprintf("expires_at <= $%d", argCount))
		args = append(args, *filters.ExpiresBefore)
	}

	return strings.Join(conditions, " AND "), args
}

func (r *PostgresContactExchangeRepository) scanContactExchangeRequest(row *sql.Row) (*domain.ContactExchangeRequest, error) {
	var id, postID, requesterUserID, ownerUserID string
	var status string
	var message *string
	var verificationRequired bool
	var verificationMethod, verificationQuestion *string
	var verificationRequirements []byte
	var approvalType, denialReason, denialMessage *string
	var encryptedContactInfo []byte
	var expiresAt, createdAt, updatedAt time.Time

	err := row.Scan(
		&id, &postID, &requesterUserID, &ownerUserID, &status, &message,
		&verificationRequired, &verificationMethod, &verificationQuestion, &verificationRequirements,
		&approvalType, &denialReason, &denialMessage, &encryptedContactInfo,
		&expiresAt, &createdAt, &updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrRepositoryNotFound("ContactExchangeRequest", id)
		}
		return nil, fmt.Errorf("failed to scan contact exchange request: %w", err)
	}

	return r.buildContactExchangeRequest(
		id, postID, requesterUserID, ownerUserID, status, message,
		verificationRequired, verificationMethod, verificationQuestion, verificationRequirements,
		approvalType, denialReason, denialMessage, encryptedContactInfo,
		expiresAt, createdAt, updatedAt,
	)
}

func (r *PostgresContactExchangeRepository) scanContactExchangeRequests(rows *sql.Rows) ([]*domain.ContactExchangeRequest, error) {
	var requests []*domain.ContactExchangeRequest

	for rows.Next() {
		var id, postID, requesterUserID, ownerUserID string
		var status string
		var message *string
		var verificationRequired bool
		var verificationMethod, verificationQuestion *string
		var verificationRequirements []byte
		var approvalType, denialReason, denialMessage *string
		var encryptedContactInfo []byte
		var expiresAt, createdAt, updatedAt time.Time

		err := rows.Scan(
			&id, &postID, &requesterUserID, &ownerUserID, &status, &message,
			&verificationRequired, &verificationMethod, &verificationQuestion, &verificationRequirements,
			&approvalType, &denialReason, &denialMessage, &encryptedContactInfo,
			&expiresAt, &createdAt, &updatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan contact exchange request: %w", err)
		}

		request, err := r.buildContactExchangeRequest(
			id, postID, requesterUserID, ownerUserID, status, message,
			verificationRequired, verificationMethod, verificationQuestion, verificationRequirements,
			approvalType, denialReason, denialMessage, encryptedContactInfo,
			expiresAt, createdAt, updatedAt,
		)

		if err != nil {
			return nil, err
		}

		requests = append(requests, request)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over contact exchange requests: %w", err)
	}

	return requests, nil
}

func (r *PostgresContactExchangeRepository) buildContactExchangeRequest(
	id, postID, requesterUserID, ownerUserID string, status string, message *string,
	verificationRequired bool, verificationMethod, verificationQuestion *string, verificationRequirements []byte,
	approvalType, denialReason, denialMessage *string, encryptedContactInfo []byte,
	expiresAt, createdAt, updatedAt time.Time,
) (*domain.ContactExchangeRequest, error) {

	requestID, err := domain.ContactExchangeRequestIDFromString(id)
	if err != nil {
		return nil, err
	}

	postUUID, err := domain.PostIDFromString(postID)
	if err != nil {
		return nil, err
	}

	requesterUUID, err := domain.UserIDFromString(requesterUserID)
	if err != nil {
		return nil, err
	}

	ownerUUID, err := domain.UserIDFromString(ownerUserID)
	if err != nil {
		return nil, err
	}

	var verificationDetails *domain.VerificationDetails
	if verificationMethod != nil {
		verificationDetails = &domain.VerificationDetails{
			Method:   domain.VerificationMethod(*verificationMethod),
			Question: verificationQuestion,
		}

		if len(verificationRequirements) > 0 {
			var requirements []string
			if err := json.Unmarshal(verificationRequirements, &requirements); err != nil {
				return nil, fmt.Errorf("failed to unmarshal verification requirements: %w", err)
			}
			verificationDetails.Requirements = requirements
		}
	}

	var parsedApprovalType *domain.ContactExchangeApprovalType
	if approvalType != nil {
		approval := domain.ContactExchangeApprovalType(*approvalType)
		parsedApprovalType = &approval
	}

	var parsedDenialReason *domain.DenialReason
	if denialReason != nil {
		denial := domain.DenialReason(*denialReason)
		parsedDenialReason = &denial
	}

	var parsedEncryptedContactInfo *domain.EncryptedContactInfo
	if len(encryptedContactInfo) > 0 {
		parsedEncryptedContactInfo = &domain.EncryptedContactInfo{}
		if err := json.Unmarshal(encryptedContactInfo, parsedEncryptedContactInfo); err != nil {
			return nil, fmt.Errorf("failed to unmarshal encrypted contact info: %w", err)
		}
	}

	return domain.ReconstructContactExchangeRequest(
		requestID,
		postUUID,
		requesterUUID,
		ownerUUID,
		domain.ContactExchangeStatus(status),
		message,
		verificationRequired,
		verificationDetails,
		parsedApprovalType,
		parsedDenialReason,
		denialMessage,
		parsedEncryptedContactInfo,
		expiresAt,
		createdAt,
		updatedAt,
	), nil
}