package domain

import (
	"fmt"
	"time"
)

// ContactExchangeStatus represents the status of a contact exchange request
type ContactExchangeStatus string

const (
	ContactExchangeStatusPending  ContactExchangeStatus = "pending"
	ContactExchangeStatusApproved ContactExchangeStatus = "approved"
	ContactExchangeStatusDenied   ContactExchangeStatus = "denied"
	ContactExchangeStatusExpired  ContactExchangeStatus = "expired"
)

// ContactExchangeApprovalType represents the type of contact sharing approved
type ContactExchangeApprovalType string

const (
	ContactExchangeApprovalTypeFull     ContactExchangeApprovalType = "full_contact"
	ContactExchangeApprovalTypePlatform ContactExchangeApprovalType = "platform_message"
	ContactExchangeApprovalTypeLimited  ContactExchangeApprovalType = "limited_contact"
)

// VerificationMethod represents required verification method
type VerificationMethod string

const (
	VerificationMethodPhotoProof      VerificationMethod = "photo_proof"
	VerificationMethodSecurityQuestion VerificationMethod = "security_question"
	VerificationMethodAdminApproval   VerificationMethod = "admin_approval"
)

// DenialReason represents reason for contact exchange denial
type DenialReason string

const (
	DenialReasonNotOwner             DenialReason = "not_owner"
	DenialReasonInsufficientVerification DenialReason = "insufficient_verification"
	DenialReasonSuspiciousRequest    DenialReason = "suspicious_request"
	DenialReasonPostResolved         DenialReason = "post_resolved"
	DenialReasonUserPreference       DenialReason = "user_preference"
	DenialReasonOther                DenialReason = "other"
)

// ContactExchangeRequest represents a secure contact exchange request
type ContactExchangeRequest struct {
	id                    ContactExchangeRequestID
	postID                PostID
	requesterUserID       UserID
	ownerUserID           UserID
	status                ContactExchangeStatus
	message               *string
	verificationRequired  bool
	verificationDetails   *VerificationDetails
	approvalType          *ContactExchangeApprovalType
	denialReason          *DenialReason
	denialMessage         *string
	encryptedContactInfo  *EncryptedContactInfo
	expiresAt             time.Time
	createdAt             time.Time
	updatedAt             time.Time
}

// VerificationDetails contains verification requirements
type VerificationDetails struct {
	Method       VerificationMethod `json:"method"`
	Question     *string           `json:"question,omitempty"`
	Requirements []string          `json:"requirements,omitempty"`
}

// EncryptedContactInfo contains encrypted contact information
type EncryptedContactInfo struct {
	Email               *string                  `json:"email,omitempty"`
	Phone               *string                  `json:"phone,omitempty"`
	PreferredMethod     string                   `json:"preferred_method"`
	Message             *string                  `json:"message,omitempty"`
	SharingRestrictions *SharingRestrictions     `json:"sharing_restrictions,omitempty"`
}

// SharingRestrictions defines limitations on contact sharing
type SharingRestrictions struct {
	ExpiresAfterHours  int  `json:"expires_after_hours"`
	SingleUse          bool `json:"single_use"`
	PlatformMediated   bool `json:"platform_mediated"`
}

// NewContactExchangeRequest creates a new contact exchange request
func NewContactExchangeRequest(
	postID PostID,
	requesterUserID UserID,
	ownerUserID UserID,
	message *string,
	verificationRequired bool,
	verificationDetails *VerificationDetails,
	expirationHours int,
) (*ContactExchangeRequest, error) {
	if postID.IsZero() {
		return nil, ErrInvalidPostID()
	}

	if requesterUserID.IsZero() || ownerUserID.IsZero() {
		return nil, ErrInvalidUserID()
	}

	if requesterUserID.Equals(ownerUserID) {
		return nil, ErrCannotRequestOwnContact()
	}

	if expirationHours <= 0 {
		expirationHours = 72 // Default 3 days
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(expirationHours) * time.Hour)

	return &ContactExchangeRequest{
		id:                   NewContactExchangeRequestID(),
		postID:               postID,
		requesterUserID:      requesterUserID,
		ownerUserID:          ownerUserID,
		status:               ContactExchangeStatusPending,
		message:              message,
		verificationRequired: verificationRequired,
		verificationDetails:  verificationDetails,
		expiresAt:            expiresAt,
		createdAt:            now,
		updatedAt:            now,
	}, nil
}

// ReconstructContactExchangeRequest reconstructs from persistence
func ReconstructContactExchangeRequest(
	id ContactExchangeRequestID,
	postID PostID,
	requesterUserID UserID,
	ownerUserID UserID,
	status ContactExchangeStatus,
	message *string,
	verificationRequired bool,
	verificationDetails *VerificationDetails,
	approvalType *ContactExchangeApprovalType,
	denialReason *DenialReason,
	denialMessage *string,
	encryptedContactInfo *EncryptedContactInfo,
	expiresAt time.Time,
	createdAt time.Time,
	updatedAt time.Time,
) *ContactExchangeRequest {
	return &ContactExchangeRequest{
		id:                   id,
		postID:               postID,
		requesterUserID:      requesterUserID,
		ownerUserID:          ownerUserID,
		status:               status,
		message:              message,
		verificationRequired: verificationRequired,
		verificationDetails:  verificationDetails,
		approvalType:         approvalType,
		denialReason:         denialReason,
		denialMessage:        denialMessage,
		encryptedContactInfo: encryptedContactInfo,
		expiresAt:            expiresAt,
		createdAt:            createdAt,
		updatedAt:            updatedAt,
	}
}

// Approve approves the contact exchange request
func (c *ContactExchangeRequest) Approve(
	approvalType ContactExchangeApprovalType,
	encryptedContactInfo *EncryptedContactInfo,
) error {
	if c.status != ContactExchangeStatusPending {
		return ErrInvalidContactExchangeStatus(c.status, ContactExchangeStatusApproved)
	}

	if c.IsExpired() {
		return ErrContactExchangeExpired()
	}

	c.status = ContactExchangeStatusApproved
	c.approvalType = &approvalType
	c.encryptedContactInfo = encryptedContactInfo
	c.updatedAt = time.Now()

	return nil
}

// Deny denies the contact exchange request
func (c *ContactExchangeRequest) Deny(reason DenialReason, message *string) error {
	if c.status != ContactExchangeStatusPending {
		return ErrInvalidContactExchangeStatus(c.status, ContactExchangeStatusDenied)
	}

	c.status = ContactExchangeStatusDenied
	c.denialReason = &reason
	c.denialMessage = message
	c.updatedAt = time.Now()

	return nil
}

// Expire marks the request as expired
func (c *ContactExchangeRequest) Expire() error {
	if c.status == ContactExchangeStatusExpired {
		return nil // Already expired
	}

	if c.status != ContactExchangeStatusPending && c.status != ContactExchangeStatusApproved {
		return ErrInvalidContactExchangeStatus(c.status, ContactExchangeStatusExpired)
	}

	c.status = ContactExchangeStatusExpired
	c.updatedAt = time.Now()

	return nil
}

// ClearContactInfo securely removes encrypted contact information
func (c *ContactExchangeRequest) ClearContactInfo() error {
	if c.status != ContactExchangeStatusExpired {
		return fmt.Errorf("can only clear contact info for expired requests")
	}

	// Securely clear the encrypted contact information
	c.encryptedContactInfo = nil
	c.updatedAt = time.Now()

	return nil
}

// IsExpired checks if the request has expired
func (c *ContactExchangeRequest) IsExpired() bool {
	return time.Now().After(c.expiresAt)
}

// CanBeApproved checks if the request can be approved
func (c *ContactExchangeRequest) CanBeApproved() bool {
	return c.status == ContactExchangeStatusPending && !c.IsExpired()
}

// CanBeDenied checks if the request can be denied
func (c *ContactExchangeRequest) CanBeDenied() bool {
	return c.status == ContactExchangeStatusPending
}

// Getters
func (c *ContactExchangeRequest) ID() ContactExchangeRequestID {
	return c.id
}

func (c *ContactExchangeRequest) PostID() PostID {
	return c.postID
}

func (c *ContactExchangeRequest) RequesterUserID() UserID {
	return c.requesterUserID
}

func (c *ContactExchangeRequest) OwnerUserID() UserID {
	return c.ownerUserID
}

func (c *ContactExchangeRequest) Status() ContactExchangeStatus {
	return c.status
}

func (c *ContactExchangeRequest) Message() *string {
	return c.message
}

func (c *ContactExchangeRequest) VerificationRequired() bool {
	return c.verificationRequired
}

func (c *ContactExchangeRequest) VerificationDetails() *VerificationDetails {
	return c.verificationDetails
}

func (c *ContactExchangeRequest) ApprovalType() *ContactExchangeApprovalType {
	return c.approvalType
}

func (c *ContactExchangeRequest) DenialReason() *DenialReason {
	return c.denialReason
}

func (c *ContactExchangeRequest) DenialMessage() *string {
	return c.denialMessage
}

func (c *ContactExchangeRequest) EncryptedContactInfo() *EncryptedContactInfo {
	return c.encryptedContactInfo
}

func (c *ContactExchangeRequest) ExpiresAt() time.Time {
	return c.expiresAt
}

func (c *ContactExchangeRequest) CreatedAt() time.Time {
	return c.createdAt
}

func (c *ContactExchangeRequest) UpdatedAt() time.Time {
	return c.updatedAt
}

// ToContactRequestData converts ContactExchangeRequest to ContactRequestData for events
func (c *ContactExchangeRequest) ToContactRequestData() ContactRequestData {
	return ContactRequestData{
		RequestID:            c.id.String(),
		Status:               string(c.status),
		Message:              c.message,
		VerificationRequired: c.verificationRequired,
		VerificationDetails:  c.verificationDetails,
		ExpiresAt:            c.expiresAt,
		CreatedAt:            c.createdAt,
	}
}