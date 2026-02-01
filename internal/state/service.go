package state

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
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
	Window    time.Duration `json:"window"` // Lookback window (default 5m)
}

// CalculateFeatures calculates features for a service from recent events
func (s *Service) CalculateFeatures(ctx context.Context, req *CalculateFeaturesRequest) (*models.ServiceFeatures, error) {
	if req.Window == 0 {
		req.Window = 5 * time.Minute
	}

	now := time.Now()
	from := now.Add(-req.Window)

	// Fetch recent metrics events for this service
	events, err := s.eventStore.GetByService(ctx, req.ServiceID, from, now, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}

	if len(events) == 0 {
		return nil, fmt.Errorf("no events found for service %s in window", req.ServiceID)
	}

	features, err := calculateFeaturesFromEvents(req.ServiceID, events)
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
			// Don't fail, just log
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

// calculateFeaturesFromEvents computes features from a list of events
func calculateFeaturesFromEvents(serviceID string, events []models.Event) (*models.ServiceFeatures, error) {
	features := &models.ServiceFeatures{
		ServiceID: serviceID,
		Timestamp: time.Now(),
	}

	var metricsList []models.MetricsPayload

	for _, evt := range events {
		if evt.EventType != models.EventTypeMetrics {
			continue
		}

		metrics, err := evt.GetMetricsPayload()
		if err != nil {
			continue
		}
		metricsList = append(metricsList, *metrics)
	}

	if len(metricsList) == 0 {
		return nil, fmt.Errorf("no metrics events found")
	}

	// Calculate aggregates
	var cpuSum, latencySum, errorRateSum, rpsSum float64
	var queueSum int
	cpuValues := make([]float64, 0, len(metricsList))
	latencyValues := make([]float64, 0, len(metricsList))

	for _, m := range metricsList {
		cpuSum += m.CPU
		cpuValues = append(cpuValues, m.CPU)

		latencySum += m.Latency
		latencyValues = append(latencyValues, m.Latency)

		errorRateSum += m.ErrorRate
		rpsSum += m.RequestsPerSec
		queueSum += m.QueueDepth
	}

	n := float64(len(metricsList))
	features.CPUCurrent = metricsList[len(metricsList)-1].CPU
	features.CPUAvg5m = cpuSum / n

	features.LatencyP50 = percentile(latencyValues, 0.5)
	features.LatencyP95 = percentile(latencyValues, 0.95)
	features.LatencyP99 = percentile(latencyValues, 0.99)

	features.ErrorRate = errorRateSum / n
	features.RequestsPerSec = rpsSum / n
	features.QueueDepth = metricsList[len(metricsList)-1].QueueDepth
	features.QueueDepthAvg5m = float64(queueSum) / n

	// Calculate EMA (Exponential Moving Average) - alpha = 0.3
	features.CPUEMA = calculateEMA(cpuValues, 0.3)
	features.LatencyEMA = calculateEMA(latencyValues, 0.3)

	// Calculate trend
	features.CPUTrend = calculateTrend(cpuValues)
	features.RequestsTrend = calculateTrend(extractRequests(metricsList))

	// Calculate composite scores
	features.LoadScore = math.Min(1.0, features.CPUCurrent/100.0*0.5+features.QueueDepthAvg5m/100.0*0.3+features.ErrorRate*0.2)
	features.HealthScore = math.Max(0, 1.0-features.ErrorRate*2-features.CPUCurrent/200.0)

	// Throttling risk (high when CPU high AND latency high)
	if features.CPUCurrent > 70 && features.LatencyP95 > 500 {
		features.ThrottlingRisk = math.Min(1.0, (features.CPUCurrent-70)/30.0*0.7+(features.LatencyP95-500)/500.0*0.3)
	}

	return features, nil
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	// Simple sort for percentile calculation
	values := make([]float64, len(sorted))
	copy(values, sorted)
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[i] > values[j] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}

	k := float64(len(values)-1) * p
	f := math.Floor(k)
	c := math.Ceil(k)
	if f == c {
		return values[int(k)]
	}
	return values[int(f)]*(c-k) + values[int(c)]*(k-f)
}

func calculateEMA(values []float64, alpha float64) float64 {
	if len(values) == 0 {
		return 0
	}
	ema := values[0]
	for i := 1; i < len(values); i++ {
		ema = alpha*values[i] + (1-alpha)*ema
	}
	return ema
}

func calculateTrend(values []float64) string {
	if len(values) < 2 {
		return "stable"
	}
	// Compare first 20% vs last 20%
	startIdx := int(float64(len(values)) * 0.2)
	endIdx := int(float64(len(values)) * 0.8)

	if startIdx == 0 {
		startIdx = 1
	}
	if endIdx >= len(values) {
		endIdx = len(values) - 1
	}

	var startSum, endSum float64
	for i := 0; i < startIdx; i++ {
		startSum += values[i]
	}
	for i := endIdx; i < len(values); i++ {
		endSum += values[i]
	}

	startAvg := startSum / float64(startIdx)
	endAvg := endSum / float64(len(values)-endIdx)

	diff := (endAvg - startAvg) / startAvg
	if diff > 0.1 {
		return "increasing"
	} else if diff < -0.1 {
		return "decreasing"
	}
	return "stable"
}

func extractRequests(metrics []models.MetricsPayload) []float64 {
	result := make([]float64, len(metrics))
	for i, m := range metrics {
		result[i] = m.RequestsPerSec
	}
	return result
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
