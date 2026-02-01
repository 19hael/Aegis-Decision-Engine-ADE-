package models

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

// ServiceFeatures contains calculated features for a service
type ServiceFeatures struct {
	ServiceID string    `json:"service_id"`
	Timestamp time.Time `json:"timestamp"`
	
	// CPU features
	CPUCurrent float64 `json:"cpu_current"`
	CPUAvg5m   float64 `json:"cpu_avg_5m"`
	CPUAvg15m  float64 `json:"cpu_avg_15m"`
	CPUEMA     float64 `json:"cpu_ema"`
	CPUTrend   string  `json:"cpu_trend"`
	
	// Latency features
	LatencyP50 float64 `json:"latency_p50"`
	LatencyP95 float64 `json:"latency_p95"`
	LatencyP99 float64 `json:"latency_p99"`
	LatencyEMA float64 `json:"latency_ema"`
	
	// Error rate features
	ErrorRate   float64 `json:"error_rate"`
	ErrorRate5m float64 `json:"error_rate_5m"`
	ErrorSpike  bool    `json:"error_spike"`
	
	// Throughput features
	RequestsPerSec   float64 `json:"requests_per_second"`
	RequestsPerSec5m float64 `json:"requests_per_second_5m"`
	RequestsTrend    string  `json:"requests_trend"`
	
	// Queue features
	QueueDepth      int     `json:"queue_depth"`
	QueueDepthAvg5m float64 `json:"queue_depth_avg_5m"`
	QueueSaturation float64 `json:"queue_saturation"`
	
	// Derived features
	LoadScore      float64 `json:"load_score"`
	HealthScore    float64 `json:"health_score"`
	ThrottlingRisk float64 `json:"throttling_risk"`
}

// FeatureSnapshot represents a persisted snapshot of features
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

// CalculateFeatures computes features from a list of events
func CalculateFeatures(serviceID string, events []Event) (*ServiceFeatures, error) {
	if len(events) == 0 {
		return nil, fmt.Errorf("no events provided")
	}
	
	features := &ServiceFeatures{
		ServiceID: serviceID,
		Timestamp: time.Now(),
	}
	
	var metricsList []MetricsPayload
	for _, evt := range events {
		if evt.EventType != EventTypeMetrics {
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
	cpuValues := make([]float64, len(metricsList))
	latencyValues := make([]float64, len(metricsList))
	
	for i, m := range metricsList {
		cpuSum += m.CPU
		cpuValues[i] = m.CPU
		
		latencySum += m.Latency
		latencyValues[i] = m.Latency
		
		errorRateSum += m.ErrorRate
		rpsSum += m.RequestsPerSec
		queueSum += m.QueueDepth
	}
	
	n := float64(len(metricsList))
	features.CPUCurrent = metricsList[len(metricsList)-1].CPU
	features.CPUAvg5m = cpuSum / n
	
	features.LatencyP50 = calculatePercentile(latencyValues, 0.5)
	features.LatencyP95 = calculatePercentile(latencyValues, 0.95)
	features.LatencyP99 = calculatePercentile(latencyValues, 0.99)
	
	features.ErrorRate = errorRateSum / n
	features.RequestsPerSec = rpsSum / n
	features.QueueDepth = metricsList[len(metricsList)-1].QueueDepth
	features.QueueDepthAvg5m = float64(queueSum) / n
	
	// Calculate EMA
	features.CPUEMA = calculateEMA(cpuValues, 0.3)
	features.LatencyEMA = calculateEMA(latencyValues, 0.3)
	
	// Calculate trends
	features.CPUTrend = calculateTrend(cpuValues)
	features.RequestsTrend = calculateTrend(extractRequests(metricsList))
	
	// Calculate composite scores
	features.LoadScore = math.Min(1.0, features.CPUCurrent/100.0*0.5+
		features.QueueDepthAvg5m/100.0*0.3+
		features.ErrorRate*0.2)
	
	features.HealthScore = math.Max(0, 1.0-
		features.ErrorRate*2-
		features.CPUCurrent/200.0)
	
	// Throttling risk
	if features.CPUCurrent > 70 && features.LatencyP95 > 500 {
		features.ThrottlingRisk = math.Min(1.0,
			(features.CPUCurrent-70)/30.0*0.7+
				(features.LatencyP95-500)/500.0*0.3)
	}
	
	return features, nil
}

func calculatePercentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	// Sort copy
	sorted := make([]float64, len(values))
	copy(sorted, values)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	k := float64(len(sorted)-1) * p
	f := math.Floor(k)
	c := math.Ceil(k)
	if f == c {
		return sorted[int(k)]
	}
	return sorted[int(f)]*(c-k) + sorted[int(c)]*(k-f)
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

func extractRequests(metrics []MetricsPayload) []float64 {
	result := make([]float64, len(metrics))
	for i, m := range metrics {
		result[i] = m.RequestsPerSec
	}
	return result
}
