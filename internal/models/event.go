package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// EventType represents the type of event
type EventType string

const (
	EventTypeMetrics EventType = "metrics"
	EventTypeAlert   EventType = "alert"
	EventTypeCustom  EventType = "custom"
)

// Event represents an incoming event to the system
type Event struct {
	ID             string          `json:"id" db:"id"`
	EventID        string          `json:"event_id" db:"event_id"`
	IdempotencyKey string          `json:"idempotency_key" db:"idempotency_key"`
	ServiceID      string          `json:"service_id" db:"service_id"`
	EventType      EventType       `json:"event_type" db:"event_type"`
	Payload        json.RawMessage `json:"payload" db:"payload"`
	Timestamp      time.Time       `json:"timestamp" db:"timestamp"`
	ProcessedAt    *time.Time      `json:"processed_at,omitempty" db:"processed_at"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
}

// Validate validates the event
func (e *Event) Validate() error {
	if e.EventID == "" {
		return NewValidationError("event_id", "event ID is required")
	}
	if e.IdempotencyKey == "" {
		return NewValidationError("idempotency_key", "idempotency key is required")
	}
	if e.ServiceID == "" {
		return NewValidationError("service_id", "service ID is required")
	}
	if e.EventType == "" {
		return NewValidationError("event_type", "event type is required")
	}
	if len(e.Payload) == 0 {
		return NewValidationError("payload", "payload is required")
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	return nil
}

// MetricsPayload represents the payload for metrics events
type MetricsPayload struct {
	CPU            float64 `json:"cpu"`
	Latency        float64 `json:"latency_ms"`
	ErrorRate      float64 `json:"error_rate"`
	RequestsPerSec float64 `json:"requests_per_second"`
	QueueDepth     int     `json:"queue_depth"`
}

// GetMetricsPayload extracts metrics from payload
func (e *Event) GetMetricsPayload() (*MetricsPayload, error) {
	if e.EventType != EventTypeMetrics {
		return nil, ErrNotMetricsEvent
	}
	
	var metrics MetricsPayload
	if err := json.Unmarshal(e.Payload, &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics payload: %w", err)
	}
	return &metrics, nil
}

// AlertPayload represents the payload for alert events
type AlertPayload struct {
	AlertType string `json:"alert_type"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
}

// GetAlertPayload extracts alert from payload
func (e *Event) GetAlertPayload() (*AlertPayload, error) {
	if e.EventType != EventTypeAlert {
		return nil, fmt.Errorf("event type is %s, not alert", e.EventType)
	}
	
	var alert AlertPayload
	if err := json.Unmarshal(e.Payload, &alert); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alert payload: %w", err)
	}
	return &alert, nil
}

// CustomPayload represents the payload for custom events
type CustomPayload struct {
	EventName string          `json:"event_name"`
	Payload   json.RawMessage `json:"payload"`
}

// GetCustomPayload extracts custom payload
func (e *Event) GetCustomPayload() (*CustomPayload, error) {
	if e.EventType != EventTypeCustom {
		return nil, fmt.Errorf("event type is %s, not custom", e.EventType)
	}
	
	var custom CustomPayload
	if err := json.Unmarshal(e.Payload, &custom); err != nil {
		return nil, fmt.Errorf("failed to unmarshal custom payload: %w", err)
	}
	return &custom, nil
}

// EventStats contains statistics about events
type EventStats struct {
	TotalCount       int64     `json:"total_count"`
	UnprocessedCount int64     `json:"unprocessed_count"`
	LastEventAt      time.Time `json:"last_event_at"`
}
