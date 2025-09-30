package service

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/jsarabia/fn-posts/internal/config"
	"github.com/jsarabia/fn-posts/internal/domain"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

// EventService handles event publishing using Confluent Cloud Kafka
type EventService struct {
	writer *kafka.Writer
	topic  string
}

// NewEventService creates a new Confluent Cloud event service
func NewEventService(cfg config.KafkaConfig) (*EventService, error) {
	// Set defaults if not provided
	batchSize := cfg.BatchSize
	if batchSize == 0 {
		batchSize = 100
	}

	batchTimeout := 10 * time.Millisecond
	if cfg.BatchTimeout != "" {
		if duration, err := time.ParseDuration(cfg.BatchTimeout); err == nil {
			batchTimeout = duration
		}
	}

	retries := cfg.Retries
	if retries == 0 {
		retries = 3
	}

	// Convert acks string to int
	acks := 1 // Default to "1"
	if cfg.Acks != "" {
		switch cfg.Acks {
		case "0":
			acks = 0
		case "1":
			acks = 1
		case "all", "-1":
			acks = -1
		default:
			if acksInt, err := strconv.Atoi(cfg.Acks); err == nil {
				acks = acksInt
			}
		}
	}

	// Create writer - configuration differs based on environment
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.BootstrapServers),
		Topic:                  cfg.Topic,
		Balancer:               &kafka.LeastBytes{},
		BatchTimeout:           batchTimeout,
		BatchSize:              batchSize,
		Async:                  false, // Synchronous for reliability
		RequiredAcks:           kafka.RequiredAcks(acks),
		AllowAutoTopicCreation: true, // Allow for local development
	}

	// Configure authentication and transport based on environment
	if cfg.APIKey != "" && cfg.APISecret != "" {
		// Confluent Cloud configuration with SASL authentication
		saslMechanism := plain.Mechanism{
			Username: cfg.APIKey,
			Password: cfg.APISecret,
		}

		// Create dialer with SASL and TLS
		dialer := &kafka.Dialer{
			Timeout:       10 * time.Second,
			DualStack:     true,
			SASLMechanism: saslMechanism,
			TLS:           &tls.Config{MinVersion: tls.VersionTLS12},
		}

		writer.Transport = &kafka.Transport{
			SASL: saslMechanism,
			TLS:  &tls.Config{MinVersion: tls.VersionTLS12},
		}

		// Set the dialer for the writer
		writer.Transport.(*kafka.Transport).Dial = dialer.DialFunc
		writer.AllowAutoTopicCreation = false // Topics should be pre-created in Confluent Cloud

		log.Printf("Kafka configured for Confluent Cloud with SASL authentication")
	} else {
		// Local development configuration without authentication
		log.Printf("Kafka configured for local development without authentication")
	}

	return &EventService{
		writer: writer,
		topic:  cfg.Topic,
	}, nil
}

// PublishEvent publishes an event to Kafka with enhanced error handling and correlation tracking
func (e *EventService) PublishEvent(ctx context.Context, event *domain.PostEvent) error {
	// Validate event before publishing
	if err := e.validateEvent(event); err != nil {
		return fmt.Errorf("event validation failed: %w", err)
	}

	// Serialize event to JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create enhanced Kafka message with comprehensive headers
	message := kafka.Message{
		Key:   []byte(event.AggregateID), // Use aggregate ID for better partitioning
		Value: eventData,
		Headers: []kafka.Header{
			{Key: "event_id", Value: []byte(event.ID.String())},
			{Key: "event_type", Value: []byte(string(event.EventType))},
			{Key: "event_version", Value: []byte(fmt.Sprintf("%d", event.EventVersion))},
			{Key: "aggregate_id", Value: []byte(event.AggregateID)},
			{Key: "aggregate_type", Value: []byte(event.AggregateType)},
			{Key: "source_service", Value: []byte(event.SourceService)},
			{Key: "timestamp", Value: []byte(event.Timestamp.Format(time.RFC3339))},
			{Key: "content_type", Value: []byte("application/json")},
			{Key: "schema_version", Value: []byte("1.0")},
		},
	}

	// Add correlation ID if present
	if event.CorrelationID != nil {
		message.Headers = append(message.Headers, kafka.Header{
			Key:   "correlation_id",
			Value: []byte(*event.CorrelationID),
		})
	}

	// Add sequence number if present
	if event.SequenceNum != nil {
		message.Headers = append(message.Headers, kafka.Header{
			Key:   "sequence_number",
			Value: []byte(fmt.Sprintf("%d", *event.SequenceNum)),
		})
	}

	// Add tenant ID if present
	if event.TenantID != nil {
		message.Headers = append(message.Headers, kafka.Header{
			Key:   "tenant_id",
			Value: []byte(event.TenantID.String()),
		})
	}

	// Add privacy level if present
	if event.Privacy != nil {
		message.Headers = append(message.Headers, kafka.Header{
			Key:   "privacy_level",
			Value: []byte(event.Privacy.PrivacyLevel),
		})
	}

	// Write message to Kafka with retries
	err = e.writeMessageWithRetry(ctx, message, 3)
	if err != nil {
		return fmt.Errorf("failed to write message to Kafka after retries: %w", err)
	}

	// Log successful publishing with correlation tracking
	correlationInfo := ""
	if event.CorrelationID != nil {
		correlationInfo = fmt.Sprintf(" [correlation_id=%s]", *event.CorrelationID)
	}

	log.Printf("Published fat event to Kafka: %s for %s %s%s",
		event.EventType, event.AggregateType, event.AggregateID, correlationInfo)

	return nil
}

// validateEvent ensures the event has all required fields for fat event processing
func (e *EventService) validateEvent(event *domain.PostEvent) error {
	if event.ID.String() == "" {
		return fmt.Errorf("event ID is required")
	}
	if event.EventType == "" {
		return fmt.Errorf("event type is required")
	}
	if event.AggregateID == "" {
		return fmt.Errorf("aggregate ID is required")
	}
	if event.AggregateType == "" {
		return fmt.Errorf("aggregate type is required")
	}
	if event.SourceService == "" {
		return fmt.Errorf("source service is required")
	}
	if event.Payload == nil {
		return fmt.Errorf("event payload is required for fat events")
	}
	return nil
}

// writeMessageWithRetry implements retry logic with exponential backoff
func (e *EventService) writeMessageWithRetry(ctx context.Context, message kafka.Message, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := e.writer.WriteMessages(ctx, message)
		if err == nil {
			return nil
		}

		lastErr = err
		if attempt < maxRetries {
			// Exponential backoff: 100ms, 200ms, 400ms
			backoff := time.Duration(100 * (1 << attempt)) * time.Millisecond
			log.Printf("Event publish attempt %d failed, retrying in %v: %v", attempt+1, backoff, err)

			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(backoff):
				continue
			}
		}
	}

	return fmt.Errorf("all %d publish attempts failed, last error: %w", maxRetries+1, lastErr)
}

// Close cleanly shuts down the Kafka writer
func (e *EventService) Close() error {
	return e.writer.Close()
}
