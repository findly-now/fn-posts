package domain

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventTypePostCreated              EventType = "post.created"
	EventTypePostUpdated              EventType = "post.updated"
	EventTypePostResolved             EventType = "post.resolved"
	EventTypePostDeleted              EventType = "post.deleted"
	EventTypePhotoAdded               EventType = "post.photo.added"
	EventTypePhotoRemoved             EventType = "post.photo.removed"
	EventTypeContactExchangeRequested EventType = "contact.exchange.requested"
	EventTypeContactExchangeApproved  EventType = "contact.exchange.approved"
	EventTypeContactExchangeDenied    EventType = "contact.exchange.denied"
	EventTypeContactExchangeExpired   EventType = "contact.exchange.expired"
)

// Complete PostEvent structure following fn-contract specification
type PostEvent struct {
	ID            uuid.UUID       `json:"id"`
	EventType     EventType       `json:"event_type"`
	EventVersion  int             `json:"event_version"`
	Timestamp     time.Time       `json:"timestamp"`
	CorrelationID *string         `json:"correlation_id,omitempty"`
	SourceService string          `json:"source_service"`
	AggregateID   string          `json:"aggregate_id"`
	AggregateType string          `json:"aggregate_type"`
	SequenceNum   *int64          `json:"sequence_number,omitempty"`
	PostID        PostID          `json:"post_id"`
	UserID        UserID          `json:"user_id"`
	TenantID      *OrganizationID `json:"tenant_id,omitempty"`
	Payload       interface{}     `json:"payload"`
	Privacy       *PrivacyContext `json:"privacy,omitempty"`
}

// PrivacyContext contains privacy-safe contact exchange information
type PrivacyContext struct {
	ContactToken     *ContactExchangeToken `json:"contact_token,omitempty"`
	ContactExpiresAt *time.Time            `json:"contact_expires_at,omitempty"`
	PrivacyLevel     string                `json:"privacy_level"`
	DataProtection   *DataProtectionInfo   `json:"data_protection,omitempty"`
}

type DataProtectionInfo struct {
	GDPRCompliant  bool     `json:"gdpr_compliant"`
	CCPACompliant  bool     `json:"ccpa_compliant"`
	DataRetention  string   `json:"data_retention"`
	EncryptionKeys []string `json:"encryption_keys,omitempty"`
}

// Fat PostCreated event data with complete context per fn-contract schema
type PostCreatedEventData struct {
	Post         PostData             `json:"post"`
	User         PrivacySafeUser      `json:"user"`
	Organization *OrganizationData    `json:"organization,omitempty"`
	AIAnalysis   *AIMetadata          `json:"ai_analysis,omitempty"`
	Triggers     *EventTriggers       `json:"triggers,omitempty"`
}

// PostData represents complete post information for events
type PostData struct {
	ID             string                 `json:"id"`
	Title          string                 `json:"title"`
	Description    *string                `json:"description,omitempty"`
	Type           string                 `json:"type"`
	Status         string                 `json:"status"`
	Location       LocationData           `json:"location"`
	RadiusMeters   int                    `json:"radius_meters"`
	Photos         []PhotoData            `json:"photos"`
	UserID         string                 `json:"user_id"`
	OrganizationID *string                `json:"organization_id,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	Metadata       *PostMetadata          `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	ResolvedAt     *time.Time             `json:"resolved_at,omitempty"`
}

type LocationData struct {
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Address   *string  `json:"address,omitempty"`
	Accuracy  *float64 `json:"accuracy,omitempty"`
	Source    *string  `json:"source,omitempty"`
}

type PhotoData struct {
	ID           string     `json:"id"`
	PostID       string     `json:"post_id"`
	OriginalURL  string     `json:"original_url"`
	ThumbnailURL *string    `json:"thumbnail_url,omitempty"`
	Filename     string     `json:"filename"`
	FileSize     int64      `json:"file_size"`
	MimeType     string     `json:"mime_type"`
	Width        int        `json:"width"`
	Height       int        `json:"height"`
	Order        int        `json:"order"`
	CreatedAt    time.Time  `json:"created_at"`
}

type PostMetadata struct {
	RewardOffered *bool                   `json:"reward_offered,omitempty"`
	RewardAmount  *float64               `json:"reward_amount,omitempty"`
	Urgency       *string                `json:"urgency,omitempty"`
	Category      *string                `json:"category,omitempty"`
	CustomFields  map[string]interface{} `json:"custom_fields,omitempty"`
}

// OrganizationData provides complete organization context
type OrganizationData struct {
	OrganizationID   string                  `json:"organization_id"`
	Name             string                  `json:"name"`
	Description      *string                 `json:"description,omitempty"`
	Type             *string                 `json:"type,omitempty"`
	Status           string                  `json:"status"`
	Settings         *OrganizationSettings   `json:"settings,omitempty"`
	Branding         *OrganizationBranding   `json:"branding,omitempty"`
	ContactPolicy    *ContactSharingPolicy   `json:"contact_policy,omitempty"`
	CreatedAt        time.Time               `json:"created_at"`
	UpdatedAt        time.Time               `json:"updated_at"`
}

type OrganizationBranding struct {
	LogoURL      *string `json:"logo_url,omitempty"`
	PrimaryColor *string `json:"primary_color,omitempty"`
	CustomDomain *string `json:"custom_domain,omitempty"`
}

// AIMetadata provides AI processing context and placeholders for fn-media-ai integration
type AIMetadata struct {
	ProcessingTriggered bool                   `json:"processing_triggered"`
	ProcessingStatus    *string                `json:"processing_status,omitempty"`
	ConfidenceScore     *float64               `json:"confidence_score,omitempty"`
	Tags                []AITag                `json:"tags,omitempty"`
	Objects             []DetectedObject       `json:"objects,omitempty"`
	Colors              []ColorAnalysis        `json:"colors,omitempty"`
	Scene               *SceneAnalysis         `json:"scene,omitempty"`
	TextContent         []ExtractedText        `json:"text_content,omitempty"`
	LocationInference   *LocationInference     `json:"location_inference,omitempty"`
	ProcessingMetrics   *ProcessingMetrics     `json:"processing_metrics,omitempty"`
	ModelVersions       map[string]string      `json:"model_versions,omitempty"`
}

type AITag struct {
	Tag        string  `json:"tag"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source"`
	Category   *string `json:"category,omitempty"`
}

type DetectedObject struct {
	Name         string              `json:"name"`
	Confidence   float64             `json:"confidence"`
	BoundingBox  *BoundingBox        `json:"bounding_box,omitempty"`
	Attributes   map[string]string   `json:"attributes,omitempty"`
}

type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type ColorAnalysis struct {
	ColorName string  `json:"color_name"`
	HexCode   *string `json:"hex_code,omitempty"`
	Confidence float64 `json:"confidence"`
	Dominant   bool   `json:"dominant"`
}

type SceneAnalysis struct {
	PrimaryScene string   `json:"primary_scene"`
	Confidence   float64  `json:"confidence"`
	SubScenes    []string `json:"sub_scenes,omitempty"`
}

type ExtractedText struct {
	Text        string       `json:"text"`
	Confidence  float64      `json:"confidence"`
	BoundingBox *BoundingBox `json:"bounding_box,omitempty"`
	Language    *string      `json:"language,omitempty"`
}

type LocationInference struct {
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Source       string  `json:"source"`
	Confidence   float64 `json:"confidence"`
	LandmarkName *string `json:"landmark_name,omitempty"`
}

type ProcessingMetrics struct {
	TotalProcessingTimeMs int                    `json:"total_processing_time_ms"`
	StartedAt             time.Time              `json:"started_at"`
	CompletedAt           time.Time              `json:"completed_at"`
	StageTimings          map[string]int         `json:"stage_timings,omitempty"`
	ResourceUsage         *ResourceUsage         `json:"resource_usage,omitempty"`
	QualityIndicators     *QualityIndicators     `json:"quality_indicators,omitempty"`
}

type ResourceUsage struct {
	CPUTimeMs    *int     `json:"cpu_time_ms,omitempty"`
	MemoryPeakMB *float64 `json:"memory_peak_mb,omitempty"`
	GPUTimeMs    *int     `json:"gpu_time_ms,omitempty"`
}

type QualityIndicators struct {
	ImageQuality         *ImageQuality         `json:"image_quality,omitempty"`
	AnalysisReliability  *AnalysisReliability  `json:"analysis_reliability,omitempty"`
}

type ImageQuality struct {
	ResolutionScore float64 `json:"resolution_score"`
	BrightnessScore float64 `json:"brightness_score"`
	ClarityScore    float64 `json:"clarity_score"`
	OverallScore    float64 `json:"overall_score"`
}

type AnalysisReliability struct {
	DetectionConsistency    float64 `json:"detection_consistency"`
	ConfidenceDistribution  string  `json:"confidence_distribution"`
	CrossValidationScore    float64 `json:"cross_validation_score"`
}

// EventTriggers specifies what downstream processing should be triggered
type EventTriggers struct {
	AIProcessing    bool `json:"ai_processing"`
	MatchProcessing bool `json:"match_processing"`
	Reindexing      bool `json:"reindexing"`
	Notifications   bool `json:"notifications"`
}

type PostUpdatedEventData struct {
	Post         PostData               `json:"post"`
	User         PrivacySafeUser        `json:"user"`
	Organization *OrganizationData      `json:"organization,omitempty"`
	Changes      map[string]interface{} `json:"changes"`
	Previous     map[string]interface{} `json:"previous"`
	UpdateReason *string                `json:"update_reason,omitempty"`
	Triggers     *EventTriggers         `json:"triggers,omitempty"`
}

type PostStatusChangedEventData struct {
	Post           PostData        `json:"post"`
	User           PrivacySafeUser `json:"user"`
	Organization   *OrganizationData `json:"organization,omitempty"`
	NewStatus      PostStatus      `json:"new_status"`
	PreviousStatus PostStatus      `json:"previous_status"`
	Reason         *string         `json:"reason,omitempty"`
	ResolvedBy     *PrivacySafeUser `json:"resolved_by,omitempty"`
	ResolutionData *ResolutionData  `json:"resolution_data,omitempty"`
}

type ResolutionData struct {
	MatchID         *string    `json:"match_id,omitempty"`
	ResolutionType  string     `json:"resolution_type"`
	SuccessMetrics  *SuccessMetrics `json:"success_metrics,omitempty"`
	ResolvedAt      time.Time  `json:"resolved_at"`
}

type SuccessMetrics struct {
	TimeToResolution     int     `json:"time_to_resolution_hours"`
	MatchAccuracy        *float64 `json:"match_accuracy,omitempty"`
	UserSatisfactionScore *int    `json:"user_satisfaction_score,omitempty"`
}

// Fat PhotoAdded event data with complete context per fn-contract schema
type PhotoAddedEventData struct {
	Post                PostData        `json:"post"`
	Photo               PhotoData       `json:"photo"`
	User                PrivacySafeUser `json:"user"`
	Organization        *OrganizationData `json:"organization,omitempty"`
	AIProcessingTrigger bool            `json:"ai_processing_trigger"`
	ProcessingPriority  *string         `json:"processing_priority,omitempty"`
	Triggers            *EventTriggers  `json:"triggers,omitempty"`
}

type PhotoRemovedEventData struct {
	Post         PostData        `json:"post"`
	Photo        PhotoData       `json:"photo"`
	User         PrivacySafeUser `json:"user"`
	Organization *OrganizationData `json:"organization,omitempty"`
	RemovalReason *string        `json:"removal_reason,omitempty"`
	Triggers      *EventTriggers `json:"triggers,omitempty"`
}

// Contact Exchange event data structures with complete fat event context
type ContactExchangeRequestedEventData struct {
	ContactRequest           ContactRequestData        `json:"contact_request"`
	RelatedPost              PostData                  `json:"related_post"`
	Requester                PrivacySafeUserExtended   `json:"requester"`
	Owner                    PrivacySafeUserExtended   `json:"owner"`
	NotificationRequirements *NotificationRequirements `json:"notification_requirements,omitempty"`
	SecurityAssessment       *SecurityAssessment       `json:"security_assessment,omitempty"`
	Organization             *OrganizationData         `json:"organization,omitempty"`
}

type ContactRequestData struct {
	RequestID            string                 `json:"request_id"`
	Status               string                 `json:"status"`
	Message              *string                `json:"message,omitempty"`
	VerificationRequired bool                   `json:"verification_required"`
	VerificationDetails  *VerificationDetails   `json:"verification_details,omitempty"`
	ExpiresAt            time.Time              `json:"expires_at"`
	CreatedAt            time.Time              `json:"created_at"`
}


type PrivacySafeUserExtended struct {
	UserID           string                `json:"user_id"`
	DisplayName      string                `json:"display_name"`
	AvatarURL        *string               `json:"avatar_url,omitempty"`
	Preferences      UserPreferences       `json:"preferences"`
	Organization     *OrganizationContext  `json:"organization,omitempty"`
	ReputationScore  *float64              `json:"reputation_score,omitempty"`
	VerificationLevel string               `json:"verification_level"`
	ContactPolicy    *ContactSharingPolicy `json:"contact_policy,omitempty"`
}

type SecurityAssessment struct {
	RiskLevel       string              `json:"risk_level"`
	RiskFactors     []string            `json:"risk_factors,omitempty"`
	TrustScore      float64             `json:"trust_score"`
	VerificationMet []string            `json:"verification_met,omitempty"`
	Recommendation  *string             `json:"recommendation,omitempty"`
}

type ContactExchangeApprovedEventData struct {
	ContactApproval          ContactApprovalData       `json:"contact_approval"`
	RelatedPost              PostData                  `json:"related_post"`
	Requester                PrivacySafeUserExtended   `json:"requester"`
	Owner                    PrivacySafeUserExtended   `json:"owner"`
	NotificationRequirements *NotificationRequirements `json:"notification_requirements,omitempty"`
	AuditTrail               *AuditTrail               `json:"audit_trail,omitempty"`
	Organization             *OrganizationData         `json:"organization,omitempty"`
	ContactInstructions      *ContactInstructions      `json:"contact_instructions,omitempty"`
}

type ContactApprovalData struct {
	RequestID            string                    `json:"request_id"`
	ApprovalType         string                    `json:"approval_type"`
	ContactToken         *EncryptedContactToken    `json:"contact_token,omitempty"`
	PreferredMethod      *string                   `json:"preferred_method,omitempty"`
	Message              *string                   `json:"message,omitempty"`
	SharingRestrictions  *SharingRestrictions      `json:"sharing_restrictions,omitempty"`
	ExpiresAt            time.Time                 `json:"expires_at"`
	ApprovedAt           time.Time                 `json:"approved_at"`
	VerificationCompleted bool                     `json:"verification_completed"`
}

type EncryptedContactToken struct {
	Token            string    `json:"token"`
	ExpiresAt        time.Time `json:"expires_at"`
	ContactMethods   []string  `json:"contact_methods"`
	SingleUse        bool      `json:"single_use"`
	PlatformMediated bool      `json:"platform_mediated"`
}


type ContactInstructions struct {
	InstructionText    string   `json:"instruction_text"`
	AvailableMethods   []string `json:"available_methods"`
	PreferredMethod    *string  `json:"preferred_method,omitempty"`
	SecurityReminders  []string `json:"security_reminders,omitempty"`
}

type ContactExchangeDeniedEventData struct {
	ContactDenial            ContactDenialData         `json:"contact_denial"`
	RelatedPost              PostData                  `json:"related_post"`
	Requester                PrivacySafeUserExtended   `json:"requester"`
	Owner                    PrivacySafeUserExtended   `json:"owner"`
	NotificationRequirements *NotificationRequirements `json:"notification_requirements,omitempty"`
	AlternativeActions       *AlternativeActions       `json:"alternative_actions,omitempty"`
	Organization             *OrganizationData         `json:"organization,omitempty"`
}

type ContactDenialData struct {
	RequestID     string    `json:"request_id"`
	DenialReason  string    `json:"denial_reason"`
	DenialMessage *string   `json:"denial_message,omitempty"`
	DeniedAt      time.Time `json:"denied_at"`
	DenialSource  string    `json:"denial_source"`
}

type AlternativeActions struct {
	SuggestedActions []string `json:"suggested_actions,omitempty"`
	RetryAllowed     bool     `json:"retry_allowed"`
	RetryAfterHours  *int     `json:"retry_after_hours,omitempty"`
	SupportContact   *string  `json:"support_contact,omitempty"`
}

type ContactExchangeExpiredEventData struct {
	ContactExpiration ContactExpirationData `json:"contact_expiration"`
	RelatedPost       PostData             `json:"related_post"`
	InvolvedUsers     InvolvedUsersExtended `json:"involved_users"`
	CleanupActions    *CleanupActions      `json:"cleanup_actions,omitempty"`
	Organization      *OrganizationData    `json:"organization,omitempty"`
	Analytics         *ExpirationAnalytics `json:"analytics,omitempty"`
}

type ContactExpirationData struct {
	RequestID        string    `json:"request_id"`
	OriginalStatus   string    `json:"original_status"`
	ExpirationReason string    `json:"expiration_reason"`
	ExpiredAt        time.Time `json:"expired_at"`
	DurationHours    float64   `json:"duration_hours"`
}

type InvolvedUsersExtended struct {
	Requester PrivacySafeUserExtended `json:"requester"`
	Owner     PrivacySafeUserExtended `json:"owner"`
}

type ExpirationAnalytics struct {
	InteractionCount   int        `json:"interaction_count"`
	ViewsCount         int        `json:"views_count"`
	LastActivity       *time.Time `json:"last_activity,omitempty"`
	ExpirationCategory string     `json:"expiration_category"`
}

// Supporting types for contact exchange events
type ContactApproval struct {
	RequestID            ContactExchangeRequestID     `json:"request_id"`
	ApprovalType         ContactExchangeApprovalType  `json:"approval_type"`
	ContactInfo          *EncryptedContactInfo        `json:"contact_info,omitempty"`
	ExpiresAt            time.Time                    `json:"expires_at"`
	ApprovedAt           time.Time                    `json:"approved_at"`
	VerificationCompleted bool                        `json:"verification_completed"`
}

type ContactDenial struct {
	RequestID      ContactExchangeRequestID `json:"request_id"`
	DenialReason   DenialReason            `json:"denial_reason"`
	DenialMessage  *string                 `json:"denial_message,omitempty"`
	DeniedAt       time.Time               `json:"denied_at"`
	DenialSource   string                  `json:"denial_source"`
}

type ContactExpiration struct {
	RequestID         ContactExchangeRequestID `json:"request_id"`
	OriginalStatus    ContactExchangeStatus   `json:"original_status"`
	ExpirationReason  string                  `json:"expiration_reason"`
	ExpiredAt         time.Time               `json:"expired_at"`
	DurationHours     float64                 `json:"duration_hours"`
}

type InvolvedUsers struct {
	Requester *PrivacySafeUser `json:"requester"`
	Owner     *PrivacySafeUser `json:"owner"`
}

type NotificationRequirements struct {
	ImmediateNotification bool                    `json:"immediate_notification"`
	NotificationTemplate  *string                `json:"notification_template,omitempty"`
	ReminderSchedule      []NotificationReminder `json:"reminder_schedule,omitempty"`
}

type NotificationReminder struct {
	DelayHours int    `json:"delay_hours"`
	Template   string `json:"template"`
	Condition  string `json:"condition,omitempty"`
}

type AuditTrail struct {
	ApprovalSource     string       `json:"approval_source"`
	ApprovalCriteriaMet []string     `json:"approval_criteria_met,omitempty"`
	RiskAssessment     *RiskAssessment `json:"risk_assessment,omitempty"`
}

type RiskAssessment struct {
	RiskLevel   string   `json:"risk_level"`
	RiskFactors []string `json:"risk_factors,omitempty"`
}

type CleanupActions struct {
	RevokeContactAccess bool `json:"revoke_contact_access"`
	ArchiveRequestData  bool `json:"archive_request_data"`
	UpdateAnalytics     bool `json:"update_analytics"`
}

func NewPostEvent(eventType EventType, postID PostID, userID UserID, tenantID *OrganizationID, payload interface{}) *PostEvent {
	return &PostEvent{
		ID:            uuid.New(),
		EventType:     eventType,
		EventVersion:  1,
		Timestamp:     time.Now(),
		SourceService: "fn-posts",
		AggregateID:   postID.String(),
		AggregateType: "Post",
		PostID:        postID,
		UserID:        userID,
		TenantID:      tenantID,
		Payload:       payload,
	}
}

func NewPostEventWithCorrelation(eventType EventType, postID PostID, userID UserID, tenantID *OrganizationID, payload interface{}, correlationID string) *PostEvent {
	event := NewPostEvent(eventType, postID, userID, tenantID, payload)
	event.CorrelationID = &correlationID
	return event
}

// Contact Exchange event constructors
func NewContactExchangeEvent(eventType EventType, requestID ContactExchangeRequestID, userID UserID, tenantID *OrganizationID, payload interface{}) *PostEvent {
	return &PostEvent{
		ID:            uuid.New(),
		EventType:     eventType,
		EventVersion:  1,
		Timestamp:     time.Now(),
		SourceService: "fn-posts",
		AggregateID:   requestID.String(),
		AggregateType: "ContactExchangeRequest",
		PostID:        PostID{}, // Not applicable for contact exchange events
		UserID:        userID,
		TenantID:      tenantID,
		Payload:       payload,
	}
}

func NewContactExchangeEventWithCorrelation(eventType EventType, requestID ContactExchangeRequestID, userID UserID, tenantID *OrganizationID, payload interface{}, correlationID string) *PostEvent {
	event := NewContactExchangeEvent(eventType, requestID, userID, tenantID, payload)
	event.CorrelationID = &correlationID
	return event
}
