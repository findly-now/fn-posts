package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jsarabia/fn-posts/internal/domain"
)

// NoOpEventPublisher for testing or when event publishing is not available
type NoOpEventPublisher struct{}

// NewNoOpEventPublisher creates a new no-op event publisher
func NewNoOpEventPublisher() *NoOpEventPublisher {
	return &NoOpEventPublisher{}
}

// PublishEvent logs the event but doesn't publish it anywhere
func (p *NoOpEventPublisher) PublishEvent(ctx context.Context, event *domain.PostEvent) error {
	fmt.Printf("NoOp: Would publish event %s for post %s\n", event.EventType, event.PostID)
	return nil
}

// Close is a no-op for the NoOpEventPublisher
func (p *NoOpEventPublisher) Close() error {
	return nil
}

// LoggingEventPublisher logs events instead of publishing (useful for development)
type LoggingEventPublisher struct{}

// NewLoggingEventPublisher creates a new logging event publisher
func NewLoggingEventPublisher() *LoggingEventPublisher {
	return &LoggingEventPublisher{}
}

// PublishEvent logs the event in a formatted way
func (p *LoggingEventPublisher) PublishEvent(ctx context.Context, event *domain.PostEvent) error {
	eventData, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal event for logging: %w", err)
	}

	fmt.Printf("Event Published:\n%s\n", string(eventData))
	return nil
}

// Close is a no-op for the LoggingEventPublisher
func (p *LoggingEventPublisher) Close() error {
	return nil
}
