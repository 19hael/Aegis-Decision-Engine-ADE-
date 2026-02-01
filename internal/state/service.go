package state

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/aegis-decision-engine/ade/internal/models"
	"github.com/aegis-decision-engine/ade/internal/storage/postgres"
)

// Service manages state and feature calculations
type Service struct {
	eventStore   *postgres.EventStore
	featureStore *postgres.FeatureStore
	logger       *slog.Logger
}

// NewService creates a new state service
func NewService(eventStore *postgres.EventStore, featureStore *postgres.FeatureStore, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		eventStore:   eventStore,
		featureStore: featureStore,
		logger:       logger,
	}
}

// CalculateFeaturesRequest represents a request to calculate features
type CalculateFeaturesRequest struct {
	ServiceID string        `json:"service_id"`
	Window    time.Duration `json:"window"`
}

// CalculateFeatures calculates features for a service from recent events
func (s *Service) CalculateFeatures(ctx context.Context, req *CalculateFeaturesRequest) (*models.ServiceFeatures, error) {
	if req.Window == 0 {
		req.Window = 5 * time.Minute
	}

	now := time.Now()
	from := now.Add(-req.Window)

	events, err := s.eventStore.GetByService(ctx, req.ServiceID, from, now, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}

	if len(events) == 0 {
		return nil, fmt.Errorf("no events found for service %s in window", req.ServiceID)
	}

	features, err := models.CalculateFeatures(req.ServiceID, events)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate features: %w", err)
	}

	// Store the snapshot
	snapshot := &models.FeatureSnapshot{
		SnapshotID:   fmt.Sprintf("snap-%s-%d", req.ServiceID, now.Unix()),
		ServiceID:    req.ServiceID,
		Features:     mustMarshal(features),
		CalculatedAt: now,
		ValidUntil:   ptr(now.Add(req.Window)),
		EventIDs:     mustMarshal(extractEventIDs(events)),
	}

	if s.featureStore != nil {
		if err := s.featureStore.Store(ctx, snapshot); err != nil {
			s.logger.Warn("failed to store feature snapshot", "error", err)
		}
	}

	s.logger.Info("features calculated",
		"service_id", req.ServiceID,
		"event_count", len(events),
		"cpu_current", features.CPUCurrent,
		"health_score", features.HealthScore,
	)

	return features, nil
}

// GetLatestFeatures retrieves the latest features for a service
func (s *Service) GetLatestFeatures(ctx context.Context, serviceID string) (*models.ServiceFeatures, error) {
	if s.featureStore == nil {
		return nil, fmt.Errorf("feature store not available")
	}
	return s.featureStore.GetServiceFeatures(ctx, serviceID)
}

// GetServiceState returns the current state of a service including features
func (s *Service) GetServiceState(ctx context.Context, serviceID string) (*ServiceState, error) {
	features, err := s.GetLatestFeatures(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	stats, err := s.eventStore.GetStats(ctx, serviceID)
	if err != nil {
		s.logger.Warn("failed to get event stats", "error", err)
		stats = &models.EventStats{}
	}

	return &ServiceState{
		ServiceID:    serviceID,
		Features:     features,
		EventStats:   stats,
		LastUpdated:  features.Timestamp,
	}, nil
}

// ServiceState represents the complete state of a service
type ServiceState struct {
	ServiceID   string                  `json:"service_id"`
	Features    *models.ServiceFeatures `json:"features"`
	EventStats  *models.EventStats      `json:"event_stats"`
	LastUpdated time.Time               `json:"last_updated"`
}

func extractEventIDs(events []models.Event) []string {
	ids := make([]string, len(events))
	for i, e := range events {
		ids[i] = e.EventID
	}
	return ids
}

func mustMarshal(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func ptr(t time.Time) *time.Time {
	return &t
}
