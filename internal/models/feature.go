package models

import (
	"encoding/json"
	"errors"
	"time"
)

// FeatureSnapshot represents calculated features for a service
type FeatureSnapshot struct {
	ID           string          `json:"id" db:"id"`
	SnapshotID   string          `json:"snapshot_id" db:"snapshot_id"`
	ServiceID    string          `json:"service_id" db:"service_id"`
	Features     json.RawMessage `json:"features" db:"features"`
	CalculatedAt time.Time       `json:"calculated_at" db:"calculated_at"`
	ValidUntil   *time.Time      `json:"valid_until,omitempty" db:"valid_until"`
	EventIDs     json.RawMessage `json:"event_ids" db:"event_ids"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
}

// ServiceFeatures contains calculated features for a service
type ServiceFeatures struct {
	ServiceID string    `json:"service_id"`
	Timestamp time.Time `json:"timestamp"`
	
	// CPU features
	CPUCurrent    float64 `json:"cpu_current"`
	CPUAvg5m      float64 `json:"cpu_avg_5m"`
	CPUAvg15m     float64 `json:"cpu_avg_15m"`
	CPUEMA        float64 `json:"cpu_ema"`
	CPUTrend      string  `json:"cpu_trend"` // increasing, decreasing, stable
	
	// Latency features
	LatencyP50    float64 `json:"latency_p50"`
	LatencyP95    float64 `json:"latency_p95"`
	LatencyP99    float64 `json:"latency_p99"`
	LatencyEMA    float64 `json:"latency_ema"`
	
	// Error rate features
	ErrorRate     float64 `json:"error_rate"`
	ErrorRate5m   float64 `json:"error_rate_5m"`
	ErrorSpike    bool    `json:"error_spike"`
	
	// Throughput features
	RequestsPerSec      float64 `json:"requests_per_second"`
	RequestsPerSec5m    float64 `json:"requests_per_second_5m"`
	RequestsTrend       string  `json:"requests_trend"`
	
	// Queue features
	QueueDepth          int     `json:"queue_depth"`
	QueueDepthAvg5m     float64 `json:"queue_depth_avg_5m"`
	QueueSaturation     float64 `json:"queue_saturation"`
	
	// Derived features
	LoadScore           float64 `json:"load_score"`      // 0-1 composite
	HealthScore         float64 `json:"health_score"`    // 0-1 composite
	ThrottlingRisk      float64 `json:"throttling_risk"` // 0-1 probability
}

// CalculateFeatures computes features from a set of events
func CalculateFeatures(serviceID string, events []Event) (*ServiceFeatures, error) {
	if len(events) == 0 {
		return nil, errors.New("no events provided")
	}
	
	// Simplified feature calculation
	features := &ServiceFeatures{
		ServiceID: serviceID,
		Timestamp: time.Now(),
	}
	
	// Calculate basic aggregates from metrics events
	var cpuSum, latencySum, errorRateSum, rpsSum float64
	var cpuCount, latencyCount, errorCount, rpsCount int
	
	for _, evt := range events {
		if evt.EventType != EventTypeMetrics {
			continue
		}
		
		metrics, err := evt.GetMetricsPayload()
		if err != nil {
			continue
		}
		
		cpuSum += metrics.CPU
		cpuCount++
		
		latencySum += metrics.Latency
		latencyCount++
		
		errorRateSum += metrics.ErrorRate
		errorCount++
		
		rpsSum += metrics.RequestsPerSec
		rpsCount++
		
		features.QueueDepth = metrics.QueueDepth
	}
	
	if cpuCount > 0 {
		features.CPUCurrent = cpuSum / float64(cpuCount)
	}
	if latencyCount > 0 {
		features.LatencyP50 = latencySum / float64(latencyCount)
	}
	if errorCount > 0 {
		features.ErrorRate = errorRateSum / float64(errorCount)
	}
	if rpsCount > 0 {
		features.RequestsPerSec = rpsSum / float64(rpsCount)
	}
	
	// Calculate simple health score
	features.HealthScore = 1.0 - (features.ErrorRate * 0.5) - (features.CPUCurrent / 200.0)
	if features.HealthScore < 0 {
		features.HealthScore = 0
	}
	
	return features, nil
}
