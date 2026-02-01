package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aegis-decision-engine/ade/internal/models"
	"github.com/jackc/pgx/v5"
)

// EventStore handles event persistence
type EventStore struct {
	client *Client
}

// NewEventStore creates a new event store
func NewEventStore(client *Client) *EventStore {
	return &EventStore{client: client}
}

// Store persists a new event
func (s *EventStore) Store(ctx context.Context, event *models.Event) error {
	query := `
		INSERT INTO events (event_id, idempotency_key, service_id, event_type, payload, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id, created_at`

	err := s.client.Pool().QueryRow(ctx, query,
		event.EventID,
		event.IdempotencyKey,
		event.ServiceID,
		event.EventType,
		event.Payload,
		event.Timestamp,
	).Scan(&event.ID, &event.CreatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			// Event already exists (idempotency key conflict)
			return nil
		}
		return fmt.Errorf("failed to store event: %w", err)
	}

	return nil
}

// GetByID retrieves an event by its ID
func (s *EventStore) GetByID(ctx context.Context, eventID string) (*models.Event, error) {
	query := `
		SELECT id, event_id, idempotency_key, service_id, event_type, payload, timestamp, processed_at, created_at
		FROM events WHERE event_id = $1`

	var event models.Event
	err := s.client.Pool().QueryRow(ctx, query, eventID).Scan(
		&event.ID, &event.EventID, &event.IdempotencyKey, &event.ServiceID,
		&event.EventType, &event.Payload, &event.Timestamp, &event.ProcessedAt, &event.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	return &event, nil
}

// GetByService retrieves events for a service within a time range
func (s *EventStore) GetByService(ctx context.Context, serviceID string, from, to time.Time, limit int) ([]models.Event, error) {
	query := `
		SELECT id, event_id, idempotency_key, service_id, event_type, payload, timestamp, processed_at, created_at
		FROM events 
		WHERE service_id = $1 AND timestamp >= $2 AND timestamp <= $3
		ORDER BY timestamp DESC
		LIMIT $4`

	rows, err := s.client.Pool().Query(ctx, query, serviceID, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	return scanEvents(rows)
}

// MarkProcessed marks an event as processed
func (s *EventStore) MarkProcessed(ctx context.Context, eventID string) error {
	query := `UPDATE events SET processed_at = NOW() WHERE event_id = $1`
	_, err := s.client.Pool().Exec(ctx, query, eventID)
	return err
}

// GetUnprocessed retrieves unprocessed events
func (s *EventStore) GetUnprocessed(ctx context.Context, limit int) ([]models.Event, error) {
	query := `
		SELECT id, event_id, idempotency_key, service_id, event_type, payload, timestamp, processed_at, created_at
		FROM events 
		WHERE processed_at IS NULL
		ORDER BY timestamp ASC
		LIMIT $1`

	rows, err := s.client.Pool().Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query unprocessed events: %w", err)
	}
	defer rows.Close()

	return scanEvents(rows)
}

func scanEvents(rows pgx.Rows) ([]models.Event, error) {
	var events []models.Event
	for rows.Next() {
		var event models.Event
		err := rows.Scan(
			&event.ID, &event.EventID, &event.IdempotencyKey, &event.ServiceID,
			&event.EventType, &event.Payload, &event.Timestamp, &event.ProcessedAt, &event.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

// EventStats contains statistics about events
type EventStats struct {
	TotalCount      int64     `json:"total_count"`
	UnprocessedCount int64    `json:"unprocessed_count"`
	LastEventAt     time.Time `json:"last_event_at"`
}

// GetStats returns statistics about events
func (s *EventStore) GetStats(ctx context.Context, serviceID string) (*EventStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE processed_at IS NULL) as unprocessed,
			MAX(timestamp) as last_event
		FROM events 
		WHERE service_id = $1`

	var stats EventStats
	var lastEventAt *time.Time
	
	err := s.client.Pool().QueryRow(ctx, query, serviceID).Scan(
		&stats.TotalCount, &stats.UnprocessedCount, &lastEventAt,
	)
	if err != nil {
		return nil, err
	}
	
	if lastEventAt != nil {
		stats.LastEventAt = *lastEventAt
	}

	return &stats, nil
}
