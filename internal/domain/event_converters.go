package domain

import (
	"time"
)

// EventConverters provides methods to convert domain objects to fat event data structures
// This ensures events contain complete context and eliminate cross-service API calls

// ToPostData converts a Post domain object to PostData for events
func (p *Post) ToPostData() PostData {
	var photos []PhotoData
	for _, photo := range p.photos {
		photos = append(photos, photo.ToPhotoData())
	}

	var orgID *string
	if p.organizationID != nil {
		id := p.organizationID.String()
		orgID = &id
	}

	var description *string
	if p.description != "" {
		description = &p.description
	}

	var resolvedAt *time.Time
	if p.status == PostStatusResolved {
		resolvedAt = &p.updatedAt
	}

	return PostData{
		ID:             p.id.String(),
		Title:          p.title,
		Description:    description,
		Type:           string(p.postType),
		Status:         string(p.status),
		Location:       p.location.ToLocationData(),
		RadiusMeters:   p.radiusMeters,
		Photos:         photos,
		UserID:         p.createdBy.String(),
		OrganizationID: orgID,
		Tags:           []string{}, // TODO: Add tags support to Post domain
		Metadata:       nil,        // TODO: Add metadata support to Post domain
		CreatedAt:      p.createdAt,
		UpdatedAt:      p.updatedAt,
		ResolvedAt:     resolvedAt,
	}
}

// ToPhotoData converts a Photo domain object to PhotoData for events
func (p *Photo) ToPhotoData() PhotoData {
	var thumbnailURL *string
	if p.thumbnailURL != "" {
		thumbnailURL = &p.thumbnailURL
	}

	return PhotoData{
		ID:           p.id.String(),
		PostID:       p.postID.String(),
		OriginalURL:  p.url,
		ThumbnailURL: thumbnailURL,
		Filename:     "", // Not stored separately in Photo
		FileSize:     p.sizeBytes,
		MimeType:     p.format,
		Width:        0, // Not stored in Photo
		Height:       0, // Not stored in Photo
		Order:        p.displayOrder,
		CreatedAt:    p.createdAt,
	}
}

// ToLocationData converts a Location domain object to LocationData for events
func (l *Location) ToLocationData() LocationData {
	return LocationData{
		Latitude:  l.Latitude,
		Longitude: l.Longitude,
		Address:   nil, // Not stored in Location
		Accuracy:  nil, // Not stored in Location
		Source:    nil, // Not stored in Location
	}
}

// ToPrivacySafeUser converts user information to privacy-safe representation for events
// Note: This method should be implemented in a user service and provided to the post service
func ToPrivacySafeUser(userID UserID, displayName string, preferences UserPreferences, organization *OrganizationContext) PrivacySafeUser {
	var avatarURL *string
	// TODO: Get avatar URL from user service

	return PrivacySafeUser{
		UserID:       userID,
		DisplayName:  displayName,
		AvatarURL:    avatarURL,
		Preferences:  preferences,
		Organization: organization,
	}
}

// ToPrivacySafeUserExtended converts user information to extended privacy-safe representation
func ToPrivacySafeUserExtended(
	userID UserID,
	displayName string,
	preferences UserPreferences,
	organization *OrganizationContext,
	reputationScore *float64,
	verificationLevel string,
	contactPolicy *ContactSharingPolicy,
) PrivacySafeUserExtended {
	var avatarURL *string
	// TODO: Get avatar URL from user service

	return PrivacySafeUserExtended{
		UserID:           userID.String(),
		DisplayName:      displayName,
		AvatarURL:        avatarURL,
		Preferences:      preferences,
		Organization:     organization,
		ReputationScore:  reputationScore,
		VerificationLevel: verificationLevel,
		ContactPolicy:    contactPolicy,
	}
}

// ToOrganizationData converts organization information to complete organization context for events
// Note: This method should be implemented in an organization service and provided to the post service
func ToOrganizationData(
	orgID OrganizationID,
	name, description, orgType, status string,
	settings *OrganizationSettings,
	branding *OrganizationBranding,
	contactPolicy *ContactSharingPolicy,
	createdAt, updatedAt time.Time,
) *OrganizationData {
	var desc *string
	if description != "" {
		desc = &description
	}

	var orgTypePtr *string
	if orgType != "" {
		orgTypePtr = &orgType
	}

	return &OrganizationData{
		OrganizationID:   orgID.String(),
		Name:             name,
		Description:      desc,
		Type:             orgTypePtr,
		Status:           status,
		Settings:         settings,
		Branding:         branding,
		ContactPolicy:    contactPolicy,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}
}

// CreateAIMetadataPlaceholder creates a placeholder AI metadata structure for new posts
// This will be populated by fn-media-ai service after processing
func CreateAIMetadataPlaceholder() *AIMetadata {
	return &AIMetadata{
		ProcessingTriggered: true,
		ProcessingStatus:    StringPtr("pending"),
		ConfidenceScore:     nil,
		Tags:                []AITag{},
		Objects:             []DetectedObject{},
		Colors:              []ColorAnalysis{},
		Scene:               nil,
		TextContent:         []ExtractedText{},
		LocationInference:   nil,
		ProcessingMetrics:   nil,
		ModelVersions:       make(map[string]string),
	}
}

// CreateEventTriggers creates appropriate triggers for different event types
func CreateEventTriggersForPostCreated() *EventTriggers {
	return &EventTriggers{
		AIProcessing:    true,
		MatchProcessing: true,
		Reindexing:      true,
		Notifications:   true,
	}
}

func CreateEventTriggersForPostUpdated() *EventTriggers {
	return &EventTriggers{
		AIProcessing:    false, // Only trigger if photos changed
		MatchProcessing: true,
		Reindexing:      true,
		Notifications:   true,
	}
}

func CreateEventTriggersForPhotoAdded() *EventTriggers {
	return &EventTriggers{
		AIProcessing:    true,
		MatchProcessing: true,
		Reindexing:      true,
		Notifications:   false, // Don't notify for photo additions
	}
}

// CreatePrivacyContext creates privacy context for events
func CreatePrivacyContext(contactToken *ContactExchangeToken, privacyLevel string) *PrivacyContext {
	var expiresAt *time.Time
	if contactToken != nil {
		expiresAt = &contactToken.ExpiresAt
	}

	return &PrivacyContext{
		ContactToken:     contactToken,
		ContactExpiresAt: expiresAt,
		PrivacyLevel:     privacyLevel,
		DataProtection: &DataProtectionInfo{
			GDPRCompliant:  true,
			CCPACompliant:  true,
			DataRetention:  "90_days",
			EncryptionKeys: []string{}, // Keys managed separately
		},
	}
}

// Helper function for string pointers
func StringPtr(s string) *string {
	return &s
}

// Helper function for float64 pointers
func Float64Ptr(f float64) *float64 {
	return &f
}

// Helper function for int pointers
func IntPtr(i int) *int {
	return &i
}

// ToPrivacySafeUserExtendedFromUser converts PrivacySafeUser to PrivacySafeUserExtended for events
func ToPrivacySafeUserExtendedFromUser(user *PrivacySafeUser) PrivacySafeUserExtended {
	// Set defaults for extended fields that are not in PrivacySafeUser
	verificationLevel := "unverified"
	var reputationScore *float64
	var contactPolicy *ContactSharingPolicy

	// Use contact policy from user preferences if available
	if user.Preferences.ContactSharingPolicy != nil {
		contactPolicy = user.Preferences.ContactSharingPolicy
	}

	return PrivacySafeUserExtended{
		UserID:           user.UserID.String(),
		DisplayName:      user.DisplayName,
		AvatarURL:        user.AvatarURL,
		Preferences:      user.Preferences,
		Organization:     user.Organization,
		ReputationScore:  reputationScore,
		VerificationLevel: verificationLevel,
		ContactPolicy:    contactPolicy,
	}
}

// ToContactApprovalData converts ContactApproval to ContactApprovalData for events
func (ca *ContactApproval) ToContactApprovalData() ContactApprovalData {
	var contactToken *EncryptedContactToken
	// Convert encrypted contact info to token if present
	// TODO: Implement proper token conversion based on business requirements

	return ContactApprovalData{
		RequestID:            ca.RequestID.String(),
		ApprovalType:         string(ca.ApprovalType),
		ContactToken:         contactToken,
		PreferredMethod:      nil, // TODO: Extract from ContactInfo if needed
		Message:              nil, // TODO: Extract from ContactInfo if needed
		SharingRestrictions:  nil, // TODO: Extract from ContactInfo if needed
		ExpiresAt:            ca.ExpiresAt,
		ApprovedAt:           ca.ApprovedAt,
		VerificationCompleted: ca.VerificationCompleted,
	}
}

// ToContactDenialData converts ContactDenial to ContactDenialData for events
func (cd *ContactDenial) ToContactDenialData() ContactDenialData {
	return ContactDenialData{
		RequestID:     cd.RequestID.String(),
		DenialReason:  string(cd.DenialReason),
		DenialMessage: cd.DenialMessage,
		DeniedAt:      cd.DeniedAt,
		DenialSource:  cd.DenialSource,
	}
}

// ToContactExpirationData converts ContactExpiration to ContactExpirationData for events
func (ce *ContactExpiration) ToContactExpirationData() ContactExpirationData {
	return ContactExpirationData{
		RequestID:        ce.RequestID.String(),
		OriginalStatus:   string(ce.OriginalStatus),
		ExpirationReason: ce.ExpirationReason,
		ExpiredAt:        ce.ExpiredAt,
		DurationHours:    ce.DurationHours,
	}
}