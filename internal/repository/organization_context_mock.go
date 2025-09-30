package repository

import (
	"context"
	"time"

	"github.com/jsarabia/fn-posts/internal/domain"
)

type MockOrganizationContextRepository struct {
	organizations map[string]*domain.OrganizationData
}

func NewMockOrganizationContextRepository() *MockOrganizationContextRepository {
	return &MockOrganizationContextRepository{
		organizations: make(map[string]*domain.OrganizationData),
	}
}

func (m *MockOrganizationContextRepository) GetOrganizationData(ctx context.Context, orgID domain.OrganizationID) (*domain.OrganizationData, error) {
	// Return a mock organization data for development
	orgType := "company"
	desc := "Mock organization for development"

	return &domain.OrganizationData{
		OrganizationID: orgID.String(),
		Name:           "Mock Organization",
		Description:    &desc,
		Type:           &orgType,
		Status:         "active",
		Settings: &domain.OrganizationSettings{
			AIEnhancementPolicy: &domain.AIEnhancementPolicy{
				AutoEnhance:         true,
				QualityThreshold:    0.8,
				NotifyOnEnhancement: true,
			},
			ContactExchangePolicy: &domain.ContactExchangePolicy{
				AutoApproveVerified:    false,
				RequireVerification:    true,
				PreferredContactMethod: "email",
			},
		},
		ContactPolicy: &domain.ContactSharingPolicy{
			AutoApproveVerified:    false,
			RequireVerification:    true,
			PreferredContactMethod: "email",
		},
		CreatedAt: time.Now().AddDate(-1, 0, 0), // Mock creation date 1 year ago
		UpdatedAt: time.Now(),
	}, nil
}

func (m *MockOrganizationContextRepository) GetOrganizationSettings(ctx context.Context, orgID domain.OrganizationID) (*domain.OrganizationSettings, error) {
	return &domain.OrganizationSettings{
		AIEnhancementPolicy: &domain.AIEnhancementPolicy{
			AutoEnhance:         true,
			QualityThreshold:    0.8,
			NotifyOnEnhancement: true,
		},
		ContactExchangePolicy: &domain.ContactExchangePolicy{
			AutoApproveVerified:    false,
			RequireVerification:    true,
			PreferredContactMethod: "email",
		},
	}, nil
}

func (m *MockOrganizationContextRepository) GetContactSharingPolicy(ctx context.Context, orgID domain.OrganizationID) (*domain.ContactSharingPolicy, error) {
	return &domain.ContactSharingPolicy{
		AutoApproveVerified:    false,
		RequireVerification:    true,
		PreferredContactMethod: "email",
	}, nil
}