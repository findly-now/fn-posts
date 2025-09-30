package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type PostID struct {
	value uuid.UUID
}

func NewPostID() PostID {
	return PostID{value: uuid.New()}
}

func PostIDFromString(s string) (PostID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return PostID{}, fmt.Errorf("invalid post ID: %w", err)
	}
	return PostID{value: id}, nil
}

func (p PostID) String() string {
	return p.value.String()
}

func (p PostID) UUID() uuid.UUID {
	return p.value
}

func (p PostID) IsZero() bool {
	return p.value == uuid.Nil
}

func (p PostID) Equals(other PostID) bool {
	return p.value == other.value
}

type UserID struct {
	value uuid.UUID
}

func NewUserID() UserID {
	return UserID{value: uuid.New()}
}

func UserIDFromString(s string) (UserID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return UserID{}, fmt.Errorf("invalid user ID: %w", err)
	}
	return UserID{value: id}, nil
}

func UserIDFromUUID(id uuid.UUID) UserID {
	return UserID{value: id}
}

func (u UserID) String() string {
	return u.value.String()
}

func (u UserID) UUID() uuid.UUID {
	return u.value
}

func (u UserID) IsZero() bool {
	return u.value == uuid.Nil
}

func (u UserID) Equals(other UserID) bool {
	return u.value == other.value
}

type OrganizationID struct {
	value uuid.UUID
}

func NewOrganizationID() OrganizationID {
	return OrganizationID{value: uuid.New()}
}

func OrganizationIDFromString(s string) (OrganizationID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return OrganizationID{}, fmt.Errorf("invalid organization ID: %w", err)
	}
	return OrganizationID{value: id}, nil
}

func OrganizationIDFromUUID(id uuid.UUID) OrganizationID {
	return OrganizationID{value: id}
}

func (o OrganizationID) String() string {
	return o.value.String()
}

func (o OrganizationID) UUID() uuid.UUID {
	return o.value
}

func (o OrganizationID) IsZero() bool {
	return o.value == uuid.Nil
}

func (o OrganizationID) Equals(other OrganizationID) bool {
	return o.value == other.value
}

type PhotoID struct {
	value uuid.UUID
}

func NewPhotoID() PhotoID {
	return PhotoID{value: uuid.New()}
}

func PhotoIDFromString(s string) (PhotoID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return PhotoID{}, fmt.Errorf("invalid photo ID: %w", err)
	}
	return PhotoID{value: id}, nil
}

func PhotoIDFromUUID(id uuid.UUID) PhotoID {
	return PhotoID{value: id}
}

func (p PhotoID) String() string {
	return p.value.String()
}

func (p PhotoID) UUID() uuid.UUID {
	return p.value
}

func (p PhotoID) IsZero() bool {
	return p.value == uuid.Nil
}

func (p PhotoID) Equals(other PhotoID) bool {
	return p.value == other.value
}

// ContactExchangeRequestID represents a unique contact exchange request identifier
type ContactExchangeRequestID struct {
	value uuid.UUID
}

func NewContactExchangeRequestID() ContactExchangeRequestID {
	return ContactExchangeRequestID{value: uuid.New()}
}

func ContactExchangeRequestIDFromString(s string) (ContactExchangeRequestID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return ContactExchangeRequestID{}, fmt.Errorf("invalid contact exchange request ID: %w", err)
	}
	return ContactExchangeRequestID{value: id}, nil
}

func ContactExchangeRequestIDFromUUID(id uuid.UUID) ContactExchangeRequestID {
	return ContactExchangeRequestID{value: id}
}

func (c ContactExchangeRequestID) String() string {
	return c.value.String()
}

func (c ContactExchangeRequestID) UUID() uuid.UUID {
	return c.value
}

func (c ContactExchangeRequestID) IsZero() bool {
	return c.value == uuid.Nil
}

func (c ContactExchangeRequestID) Equals(other ContactExchangeRequestID) bool {
	return c.value == other.value
}

// PrivacySafeUser represents user context without PII for event publishing
type PrivacySafeUser struct {
	UserID       UserID                  `json:"user_id"`
	DisplayName  string                  `json:"display_name"`
	AvatarURL    *string                 `json:"avatar_url,omitempty"`
	Preferences  UserPreferences         `json:"preferences"`
	Organization *OrganizationContext    `json:"organization_context,omitempty"`
}

// UserPreferences contains notification and display preferences
type UserPreferences struct {
	Timezone             string                   `json:"timezone"`
	Language             string                   `json:"language"`
	NotificationChannels []NotificationChannel    `json:"notification_channels"`
	QuietHours           *QuietHours              `json:"quiet_hours,omitempty"`
	ContactSharingPolicy *ContactSharingPolicy    `json:"contact_sharing_policy,omitempty"`
}

// OrganizationContext provides organization context without exposing sensitive data
type OrganizationContext struct {
	OrganizationID   OrganizationID       `json:"organization_id"`
	OrganizationName string               `json:"organization_name"`
	Role             OrganizationRole     `json:"role"`
	Settings         *OrganizationSettings `json:"settings,omitempty"`
}

// OrganizationSettings contains policies and configurations
type OrganizationSettings struct {
	AIEnhancementPolicy *AIEnhancementPolicy `json:"ai_enhancement_policy,omitempty"`
	ContactExchangePolicy *ContactExchangePolicy `json:"contact_exchange_policy,omitempty"`
}

// AIEnhancementPolicy defines organization's AI enhancement settings
type AIEnhancementPolicy struct {
	AutoEnhance         bool    `json:"auto_enhance"`
	QualityThreshold    float64 `json:"quality_threshold"`
	NotifyOnEnhancement bool    `json:"notify_on_enhancement"`
}

// ContactExchangePolicy defines organization's contact sharing policies
type ContactExchangePolicy struct {
	AutoApproveVerified    bool   `json:"auto_approve_verified"`
	RequireVerification    bool   `json:"require_verification"`
	PreferredContactMethod string `json:"preferred_contact_method"`
}

// NotificationChannel represents available notification channels
type NotificationChannel string

const (
	NotificationChannelEmail    NotificationChannel = "email"
	NotificationChannelSMS      NotificationChannel = "sms"
	NotificationChannelWhatsApp NotificationChannel = "whatsapp"
	NotificationChannelPush     NotificationChannel = "push"
)

// OrganizationRole represents user's role in organization
type OrganizationRole string

const (
	OrganizationRoleAdmin  OrganizationRole = "admin"
	OrganizationRoleStaff  OrganizationRole = "staff"
	OrganizationRoleViewer OrganizationRole = "viewer"
)

// QuietHours defines when user doesn't want notifications
type QuietHours struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// ContactSharingPolicy defines user's contact sharing preferences
type ContactSharingPolicy struct {
	AutoApproveVerified    bool   `json:"auto_approve_verified"`
	RequireVerification    bool   `json:"require_verification"`
	PreferredContactMethod string `json:"preferred_contact_method"`
}

// ContactExchangeToken represents secure encrypted contact information
type ContactExchangeToken struct {
	Token            string    `json:"token"`
	ExpiresAt        time.Time `json:"expires_at"`
	ContactMethods   []string  `json:"contact_methods"`
	SingleUse        bool      `json:"single_use"`
	PlatformMediated bool      `json:"platform_mediated"`
}
