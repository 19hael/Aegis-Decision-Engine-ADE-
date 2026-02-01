package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/aegis-decision-engine/ade/internal/models"
	"github.com/aegis-decision-engine/ade/internal/storage/postgres"
	"github.com/segmentio/kafka-go"
)

// Service handles event ingestion
type Service struct {
	eventStore *postgres.EventStore
	kafkaWriter *kafka.Writer
	logger     *slog.Logger
}

// NewService creates a new ingest service
func NewService(eventStore *postgres.EventStore, kafkaWriter *kafka.Writer, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		eventStore:  eventStore,
		kafkaWriter: kafkaWriter,
		logger:      logger,
	}
}

// IngestRequest represents a request to ingest an event
type IngestRequest struct {
	EventID        string          `json:"event_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	ServiceID      string          `json:"service_id"`
	EventType      models.EventType `json:"event_type"`
	Payload        json.RawMessage `json:"payload"`
	Timestamp      time.Time       `json:"timestamp"`
}

// Validate validates the ingest request
func (r *IngestRequest) Validate() error {
	if r.EventID == "" {
		return fmt.Errorf("event_id is required")
	}
	if r.IdempotencyKey == "" {
		return fmt.Errorf("idempotency_key is required")
	}
	if r.ServiceID == "" {
		return fmt.Errorf("service_id is required")
	}
	if r.EventType == "" {
		return fmt.Errorf("event_type is required")
	}
	if len(r.Payload) == 0 {
		return fmt.Errorf("payload is required")
	}
	if r.Timestamp.IsZero() {
		r.Timestamp = time.Now()
	}
	return nil
}

// IngestResponse represents the response from ingestion
type IngestResponse struct {
	EventID   string    `json:"event_id"`
	Status    string    `json:"status"`
	Stored    bool      `json:"stored"`
	Published bool      `json:"published"`
	Timestamp time.Time `json:"timestamp"`
}

// Ingest processes an incoming event
func (s *Service) Ingest(ctx context.Context, req *IngestRequest) (*IngestResponse, error) {
	start := time.Now()
	
	if err := req.Validate(); err != nil {
		s.logger.Error("validation failed", "error", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	event := &models.Event{
		EventID:        req.EventID,
		IdempotencyKey: req.IdempotencyKey,
		ServiceID:      req.ServiceID,
		EventType:      req.EventType,
		Payload:        req.Payload,
		Timestamp:      req.Timestamp,
	}

	// Store in database
	if err := s.eventStore.Store(ctx, event); err != nil {
		s.logger.Error("failed to store event", "error", err, "event_id", req.EventID)
		return nil, fmt.Errorf("failed to store event: %w", err)
	}

	// Publish to Kafka
	published := false
	if s.kafkaWriter != nil {
		if err := s.publishToKafka(ctx, event); err != nil {
			s.logger.Warn("failed to publish to kafka", "error", err, "event_id", req.EventID)
		} else {
			published = true
		}
	}

	s.logger.Info("event ingested", 
		"event_id", req.EventID, 
		"service_id", req.ServiceID,
		"duration_ms", time.Since(start).Milliseconds(),
		"published", published,
	)

	return &IngestResponse{
		EventID:   req.EventID,
		Status:    "accepted",
		Stored:    true,
		Published: published,
		Timestamp: time.Now(),
	}, nil
}

func (s *Service) publishToKafka(ctx context.Context, event *models.Event) error {
	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(event.ServiceID),
		Value: value,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(event.EventType)},
			{Key: "service_id", Value: []byte(event.ServiceID)},
		},
	}

	return s.kafkaWriter.WriteMessages(ctx, msg)
}

// IngestBatch processes multiple events
func (s *Service) IngestBatch(ctx context.Context, requests []*IngestRequest) ([]*IngestResponse, error) {
	responses := make([]*IngestResponse, 0, len(requests))
	
	for _, req := range requests {
		resp, err := s.Ingest(ctx, req)
		if err != nil {
			s.logger.Error("batch ingest failed for event", "error", err, "event_id", req.EventID)
			// Continue with other events, don't fail the whole batch
			resp = &IngestResponse{
				EventID:   req.EventID,
				Status:    "error",
				Stored:    false,
				Published: false,
				Timestamp: time.Now(),
			}
		}
		responses = append(responses, resp)
	}
	
	return responses, nil
}
