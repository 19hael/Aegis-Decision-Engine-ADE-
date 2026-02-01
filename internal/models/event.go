package models

import (
	"encoding/json"
	"time"
)

// EventType represents the type of event
type EventType string

const (
	EventTypeMetrics EventType = "metrics"
	EventTypeAlert   EventType = "alert"
	EventTypeCustom  EventType = "custom"
)

// MetricsPayload represents the payload for metrics events
type MetricsPayload struct {
	CPU             float64 `json:"cpu"`
	Latency         float64 `json:"latency_ms"`
	ErrorRate       float64 `json:"error_rate"`
	RequestsPerSec  float64 `json:"requests_per_second"`
	QueueDepth      int     `json:"queue_depth"`
}

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

// Validate checks if the event is valid
func (e *Event) Validate() error {
	if e.EventID == "" {
		return ErrInvalidEventID
	}
	if e.IdempotencyKey == "" {
		return ErrMissingIdempotencyKey
	}
	if e.ServiceID == "" {
		return ErrMissingServiceID
	}
	if e.EventType == "" {
		return ErrMissingEventType
	}
	if len(e.Payload) == 0 {
		return ErrMissingPayload
	}
	return nil
}

// GetMetricsPayload extracts metrics from payload
func (e *Event) GetMetricsPayload() (*MetricsPayload, error) {
	if e.EventType != EventTypeMetrics {
		return nil, ErrNotMetricsEvent
	}
	
	var metrics MetricsPayload
	if err := json.Unmarshal(e.Payload, &metrics); err != nil {
		return nil, err
	}
	return &metrics, nil
}
