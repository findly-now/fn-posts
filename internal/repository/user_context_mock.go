package repository

import (
	"context"
	"fmt"

	"github.com/jsarabia/fn-posts/internal/domain"
)

// MockUserContextRepository provides mock privacy-safe user context
// This should be replaced with actual user service integration following privacy requirements
type MockUserContextRepository struct {
	users map[domain.UserID]*domain.PrivacySafeUser
}

func NewMockUserContextRepository() *MockUserContextRepository {
	return &MockUserContextRepository{
		users: make(map[domain.UserID]*domain.PrivacySafeUser),
	}
}

// GetPrivacySafeUser returns privacy-safe user context
// TODO: Replace with actual implementation that fetches from user service or local cache
func (r *MockUserContextRepository) GetPrivacySafeUser(ctx context.Context, userID domain.UserID) (*domain.PrivacySafeUser, error) {
	if user, exists := r.users[userID]; exists {
		return user, nil
	}

	// Mock implementation - create default privacy-safe user
	// In production, this would fetch from user service via events or API calls
	user := &domain.PrivacySafeUser{
		UserID:      userID,
		DisplayName: fmt.Sprintf("User %s", userID.String()[:8]),
		Preferences: domain.UserPreferences{
			Timezone:             "UTC",
			Language:             "en",
			NotificationChannels: []domain.NotificationChannel{domain.NotificationChannelEmail},
		},
	}

	// Cache the mock user
	r.users[userID] = user
	return user, nil
}

func (r *MockUserContextRepository) GetPrivacySafeUsers(ctx context.Context, userIDs []domain.UserID) (map[domain.UserID]*domain.PrivacySafeUser, error) {
	result := make(map[domain.UserID]*domain.PrivacySafeUser)

	for _, userID := range userIDs {
		user, err := r.GetPrivacySafeUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		result[userID] = user
	}

	return result, nil
}

// SetMockUser allows setting mock user data for testing
func (r *MockUserContextRepository) SetMockUser(userID domain.UserID, user *domain.PrivacySafeUser) {
	r.users[userID] = user
}

// ClearMockUsers clears all mock users
func (r *MockUserContextRepository) ClearMockUsers() {
	r.users = make(map[domain.UserID]*domain.PrivacySafeUser)
}