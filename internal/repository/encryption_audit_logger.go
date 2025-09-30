package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jsarabia/fn-posts/internal/domain"
)

type PostgresEncryptionAuditLogger struct {
	db *sql.DB
}

func NewPostgresEncryptionAuditLogger(db *sql.DB) *PostgresEncryptionAuditLogger {
	return &PostgresEncryptionAuditLogger{db: db}
}

func (l *PostgresEncryptionAuditLogger) LogOperation(log *domain.EncryptionAuditLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}

	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	query := `
		INSERT INTO encryption_audit_logs (
			id, operation, user_id, request_id, key_fingerprint, success,
			error_message, timestamp, ip_address, user_agent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	var requestID *string
	if log.RequestID != nil {
		requestIDStr := log.RequestID.String()
		requestID = &requestIDStr
	}

	_, err := l.db.Exec(
		query,
		log.ID,
		string(log.Operation),
		log.UserID.String(),
		requestID,
		log.KeyFingerprint,
		log.Success,
		log.ErrorMessage,
		log.Timestamp,
		log.IPAddress,
		log.UserAgent,
	)

	if err != nil {
		return fmt.Errorf("failed to log encryption operation: %w", err)
	}

	return nil
}

func (l *PostgresEncryptionAuditLogger) GetAuditTrail(userID domain.UserID, requestID *domain.ContactExchangeRequestID, limit int) ([]*domain.EncryptionAuditLog, error) {
	if limit <= 0 {
		limit = 100
	}

	baseQuery := `
		SELECT id, operation, user_id, request_id, key_fingerprint, success,
			   error_message, timestamp, ip_address, user_agent
		FROM encryption_audit_logs
		WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

	if !userID.IsZero() {
		baseQuery += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, userID.String())
		argIndex++
	}

	if requestID != nil {
		baseQuery += fmt.Sprintf(" AND request_id = $%d", argIndex)
		args = append(args, requestID.String())
		argIndex++
	}

	baseQuery += fmt.Sprintf(" ORDER BY timestamp DESC LIMIT $%d", argIndex)
	args = append(args, limit)

	rows, err := l.db.Query(baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit trail: %w", err)
	}
	defer rows.Close()

	var logs []*domain.EncryptionAuditLog

	for rows.Next() {
		var log domain.EncryptionAuditLog
		var operation string
		var userIDStr string
		var requestIDStr sql.NullString
		var errorMessage sql.NullString
		var ipAddress sql.NullString
		var userAgent sql.NullString

		err := rows.Scan(
			&log.ID,
			&operation,
			&userIDStr,
			&requestIDStr,
			&log.KeyFingerprint,
			&log.Success,
			&errorMessage,
			&log.Timestamp,
			&ipAddress,
			&userAgent,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		// Convert operation string to enum
		log.Operation = domain.EncryptionOperation(operation)

		// Parse user ID
		userUUID, err := uuid.Parse(userIDStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse user ID: %w", err)
		}
		log.UserID = domain.UserIDFromUUID(userUUID)

		// Parse request ID if present
		if requestIDStr.Valid {
			requestUUID, err := uuid.Parse(requestIDStr.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse request ID: %w", err)
			}
			requestID := domain.ContactExchangeRequestIDFromUUID(requestUUID)
			log.RequestID = &requestID
		}

		// Set optional fields
		if errorMessage.Valid {
			log.ErrorMessage = &errorMessage.String
		}
		if ipAddress.Valid {
			log.IPAddress = &ipAddress.String
		}
		if userAgent.Valid {
			log.UserAgent = &userAgent.String
		}

		logs = append(logs, &log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit logs: %w", err)
	}

	return logs, nil
}

// LogEncryptionSuccess logs a successful encryption operation
func (l *PostgresEncryptionAuditLogger) LogEncryptionSuccess(userID domain.UserID, requestID *domain.ContactExchangeRequestID, keyFingerprint string, ipAddress, userAgent *string) error {
	log := &domain.EncryptionAuditLog{
		Operation:      domain.EncryptionOperationEncrypt,
		UserID:         userID,
		RequestID:      requestID,
		KeyFingerprint: keyFingerprint,
		Success:        true,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
	}
	return l.LogOperation(log)
}

// LogDecryptionSuccess logs a successful decryption operation
func (l *PostgresEncryptionAuditLogger) LogDecryptionSuccess(userID domain.UserID, requestID *domain.ContactExchangeRequestID, keyFingerprint string, ipAddress, userAgent *string) error {
	log := &domain.EncryptionAuditLog{
		Operation:      domain.EncryptionOperationDecrypt,
		UserID:         userID,
		RequestID:      requestID,
		KeyFingerprint: keyFingerprint,
		Success:        true,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
	}
	return l.LogOperation(log)
}

// LogEncryptionFailure logs a failed encryption operation
func (l *PostgresEncryptionAuditLogger) LogEncryptionFailure(userID domain.UserID, requestID *domain.ContactExchangeRequestID, keyFingerprint string, errorMsg string, ipAddress, userAgent *string) error {
	log := &domain.EncryptionAuditLog{
		Operation:      domain.EncryptionOperationEncrypt,
		UserID:         userID,
		RequestID:      requestID,
		KeyFingerprint: keyFingerprint,
		Success:        false,
		ErrorMessage:   &errorMsg,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
	}
	return l.LogOperation(log)
}

// LogDecryptionFailure logs a failed decryption operation
func (l *PostgresEncryptionAuditLogger) LogDecryptionFailure(userID domain.UserID, requestID *domain.ContactExchangeRequestID, keyFingerprint string, errorMsg string, ipAddress, userAgent *string) error {
	log := &domain.EncryptionAuditLog{
		Operation:      domain.EncryptionOperationDecrypt,
		UserID:         userID,
		RequestID:      requestID,
		KeyFingerprint: keyFingerprint,
		Success:        false,
		ErrorMessage:   &errorMsg,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
	}
	return l.LogOperation(log)
}

// LogKeyRotation logs a key rotation operation
func (l *PostgresEncryptionAuditLogger) LogKeyRotation(userID domain.UserID, oldKeyFingerprint, newKeyFingerprint string, success bool, errorMsg *string, ipAddress, userAgent *string) error {
	log := &domain.EncryptionAuditLog{
		Operation:      domain.EncryptionOperationKeyRotation,
		UserID:         userID,
		KeyFingerprint: fmt.Sprintf("old:%s,new:%s", oldKeyFingerprint, newKeyFingerprint),
		Success:        success,
		ErrorMessage:   errorMsg,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
	}
	return l.LogOperation(log)
}