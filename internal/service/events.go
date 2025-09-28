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

	// Configure SASL authentication for Confluent Cloud
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

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.BootstrapServers),
		Topic:                  cfg.Topic,
		Balancer:               &kafka.LeastBytes{},
		BatchTimeout:           batchTimeout,
		BatchSize:              batchSize,
		Async:                  false, // Synchronous for reliability
		RequiredAcks:           kafka.RequiredAcks(acks),
		AllowAutoTopicCreation: false, // Topics should be pre-created in Confluent Cloud
		Transport: &kafka.Transport{
			SASL: saslMechanism,
			TLS:  &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}

	// Set the dialer for the writer
	writer.Transport.(*kafka.Transport).Dial = dialer.DialFunc

	return &EventService{
		writer: writer,
		topic:  cfg.Topic,
	}, nil
}

// PublishEvent publishes an event to Confluent Cloud
func (e *EventService) PublishEvent(ctx context.Context, event *domain.PostEvent) error {
	// Serialize event to JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create Kafka message with enhanced headers for Confluent Cloud
	message := kafka.Message{
		Key:   []byte(event.PostID.String()), // Partition by post ID
		Value: eventData,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(event.EventType)},
			{Key: "post_id", Value: []byte(event.PostID.String())},
			{Key: "user_id", Value: []byte(event.UserID.String())},
			{Key: "timestamp", Value: []byte(event.Timestamp.Format(time.RFC3339))},
			{Key: "content_type", Value: []byte("application/json")},
			{Key: "producer", Value: []byte("posts-service")},
			{Key: "version", Value: []byte("1.0")},
		},
	}

	if event.TenantID != nil {
		message.Headers = append(message.Headers, kafka.Header{
			Key:   "tenant_id",
			Value: []byte(event.TenantID.String()),
		})
	}

	// Write message to Confluent Cloud
	err = e.writer.WriteMessages(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to write message to Confluent Cloud: %w", err)
	}

	log.Printf("Published event to Confluent Cloud: %s for post %s", event.EventType, event.PostID)
	return nil
}

// Close cleanly shuts down the Confluent Cloud writer
func (e *EventService) Close() error {
	return e.writer.Close()
}
