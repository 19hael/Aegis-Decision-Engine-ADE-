package feedback

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"
)

// Service handles feedback and drift detection
type Service struct {
	logger *slog.Logger
}

// NewService creates a new feedback service
func NewService(logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		logger: logger,
	}
}

// FeedbackRequest represents a request to record feedback
type FeedbackRequest struct {
	ActionID                string                  `json:"action_id"`
	DecisionID              string                  `json:"decision_id"`
	ServiceID               string                  `json:"service_id"`
	FeedbackType            string                  `json:"feedback_type"` // immediate, delayed, scheduled
	MetricsBefore           map[string]float64      `json:"metrics_before"`
	MetricsAfter            map[string]float64      `json:"metrics_after"`
	ObservationWindowMins   int                     `json:"observation_window_minutes"`
}

// FeedbackResult represents the result of feedback analysis
type FeedbackResult struct {
	FeedbackID           string                 `json:"feedback_id"`
	ActionID             string                 `json:"action_id"`
	DecisionID           string                 `json:"decision_id"`
	ServiceID            string                 `json:"service_id"`
	ImpactScore          float64                `json:"impact_score"` // -1 to 1
	DriftDetected        bool                   `json:"drift_detected"`
	DriftDetails         DriftDetails           `json:"drift_details,omitempty"`
	RollbackRecommended  bool                   `json:"rollback_recommended"`
	RollbackExecuted     bool                   `json:"rollback_executed"`
	RecordedAt           time.Time              `json:"recorded_at"`
}

// DriftDetails contains drift detection information
type DriftDetails struct {
	DriftType        string             `json:"drift_type"`
	Severity         string             `json:"severity"` // low, medium, high, critical
	MetricsDrifted   []MetricDrift      `json:"metrics_drifted"`
	ThresholdViolated float64           `json:"threshold_violated,omitempty"`
	Description      string             `json:"description"`
}

// MetricDrift represents drift in a specific metric
type MetricDrift struct {
	Metric       string  `json:"metric"`
	Before       float64 `json:"before"`
	After        float64 `json:"after"`
	ChangePct    float64 `json:"change_pct"`
	DriftDetected bool   `json:"drift_detected"`
}

// RecordFeedback records and analyzes feedback
func (s *Service) RecordFeedback(ctx context.Context, req *FeedbackRequest) (*FeedbackResult, error) {
	if err := s.validateRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if req.ObservationWindowMins == 0 {
		req.ObservationWindowMins = 5
	}

	feedbackID := fmt.Sprintf("fbk-%d", time.Now().UnixNano())

	// Calculate impact score
	impactScore := s.calculateImpactScore(req.MetricsBefore, req.MetricsAfter)

	// Detect drift
	drift, driftDetected := s.detectDrift(req.MetricsBefore, req.MetricsAfter)

	// Determine if rollback is recommended
	rollbackRecommended := s.shouldRollback(impactScore, driftDetected, drift)

	result := &FeedbackResult{
		FeedbackID:          feedbackID,
		ActionID:            req.ActionID,
		DecisionID:          req.DecisionID,
		ServiceID:           req.ServiceID,
		ImpactScore:         impactScore,
		DriftDetected:       driftDetected,
		DriftDetails:        drift,
		RollbackRecommended: rollbackRecommended,
		RollbackExecuted:    false, // Would be set by rollback logic
		RecordedAt:          time.Now(),
	}

	s.logger.Info("feedback recorded",
		"feedback_id", feedbackID,
		"action_id", req.ActionID,
		"impact_score", impactScore,
		"drift_detected", driftDetected,
		"rollback_recommended", rollbackRecommended,
	)

	return result, nil
}

// RollbackRequest represents a request to rollback an action
type RollbackRequest struct {
	ActionID      string `json:"action_id"`
	DecisionID    string `json:"decision_id"`
	ServiceID     string `json:"service_id"`
	Reason        string `json:"reason"`
	Force         bool   `json:"force"` // Force rollback even if not recommended
}

// RollbackResult represents the result of a rollback
type RollbackResult struct {
	RollbackID    string    `json:"rollback_id"`
	ActionID      string    `json:"action_id"`
	Status        string    `json:"status"`
	Reason        string    `json:"reason"`
	ExecutedAt    time.Time `json:"executed_at"`
}

// Rollback executes a rollback
func (s *Service) Rollback(ctx context.Context, req *RollbackRequest) (*RollbackResult, error) {
	if req.ActionID == "" {
		return nil, fmt.Errorf("action_id is required")
	}
	if req.ServiceID == "" {
		return nil, fmt.Errorf("service_id is required")
	}

	rollbackID := fmt.Sprintf("rbk-%d", time.Now().UnixNano())

	s.logger.Info("executing rollback",
		"rollback_id", rollbackID,
		"action_id", req.ActionID,
		"reason", req.Reason,
		"force", req.Force,
	)

	// In a real implementation, this would:
	// 1. Look up the original action
	// 2. Execute the inverse action (e.g., scale_down if original was scale_up)
	// 3. Record the rollback

	return &RollbackResult{
		RollbackID: rollbackID,
		ActionID:   req.ActionID,
		Status:     "executed",
		Reason:     req.Reason,
		ExecutedAt: time.Now(),
	}, nil
}

func (s *Service) validateRequest(req *FeedbackRequest) error {
	if req.ActionID == "" {
		return fmt.Errorf("action_id is required")
	}
	if req.DecisionID == "" {
		return fmt.Errorf("decision_id is required")
	}
	if req.ServiceID == "" {
		return fmt.Errorf("service_id is required")
	}
	if len(req.MetricsBefore) == 0 {
		return fmt.Errorf("metrics_before is required")
	}
	if len(req.MetricsAfter) == 0 {
		return fmt.Errorf("metrics_after is required")
	}
	return nil
}

func (s *Service) calculateImpactScore(before, after map[string]float64) float64 {
	if len(before) == 0 || len(after) == 0 {
		return 0
	}

	var totalImpact float64
	var metricCount int

	// Weight different metrics
	weights := map[string]float64{
		"cpu":        0.25,
		"latency":    0.25,
		"error_rate": 0.30,
		"throughput": 0.20,
	}

	for metric, beforeVal := range before {
		afterVal, ok := after[metric]
		if !ok {
			continue
		}

		weight := weights[metric]
		if weight == 0 {
			weight = 0.1 // Default weight
		}

		// Calculate percent change
		var changePct float64
		if beforeVal != 0 {
			changePct = (afterVal - beforeVal) / beforeVal
		} else if afterVal > 0 {
			changePct = 1.0
		}

		// Normalize impact: negative change is bad (degradation)
		// For error_rate and latency, lower is better
		// For throughput, higher is better
		impact := -changePct * weight
		if metric == "throughput" || metric == "rps" {
			impact = changePct * weight // Reverse for throughput
		}

		totalImpact += impact
		metricCount++
	}

	if metricCount == 0 {
		return 0
	}

	// Normalize to -1 to 1 range
	score := totalImpact / float64(metricCount) * 5 // Scale up
	return math.Max(-1, math.Min(1, score))
}

func (s *Service) detectDrift(before, after map[string]float64) (DriftDetails, bool) {
	details := DriftDetails{
		MetricsDrifted: make([]MetricDrift, 0),
	}

	driftThreshold := 0.20 // 20% change is considered drift
	criticalThreshold := 0.50

	hasDrift := false
	criticalCount := 0
	var maxChange float64

	driftMetrics := []string{"cpu", "latency", "error_rate", "throughput", "memory"}

	for _, metric := range driftMetrics {
		beforeVal, ok1 := before[metric]
		afterVal, ok2 := after[metric]
		if !ok1 || !ok2 {
			continue
		}

		changePct := math.Abs(afterVal - beforeVal)
		if beforeVal != 0 {
			changePct = changePct / beforeVal
		}

		drift := MetricDrift{
			Metric:    metric,
			Before:    beforeVal,
			After:     afterVal,
			ChangePct: changePct,
		}

		if changePct > driftThreshold {
			drift.DriftDetected = true
			details.MetricsDrifted = append(details.MetricsDrifted, drift)
			hasDrift = true

			if changePct > criticalThreshold {
				criticalCount++
			}
			if changePct > maxChange {
				maxChange = changePct
			}
		}
	}

	if !hasDrift {
		return details, false
	}

	// Determine drift type and severity
	details.DriftType = s.classifyDriftType(details.MetricsDrifted)
	details.ThresholdViolated = maxChange

	if criticalCount >= 2 || maxChange > 0.8 {
		details.Severity = "critical"
	} else if criticalCount == 1 || maxChange > 0.5 {
		details.Severity = "high"
	} else if len(details.MetricsDrifted) >= 2 {
		details.Severity = "medium"
	} else {
		details.Severity = "low"
	}

	details.Description = fmt.Sprintf(
		"Detected drift in %d metrics, max change: %.1f%%, severity: %s",
		len(details.MetricsDrifted), maxChange*100, details.Severity,
	)

	return details, true
}

func (s *Service) classifyDriftType(drifts []MetricDrift) string {
	if len(drifts) == 0 {
		return "none"
	}

	hasPerformance := false
	hasError := false
	hasResource := false

	for _, d := range drifts {
		switch d.Metric {
		case "latency", "throughput":
			hasPerformance = true
		case "error_rate":
			hasError = true
		case "cpu", "memory":
			hasResource = true
		}
	}

	if hasError {
		return "error_drift"
	}
	if hasPerformance {
		return "performance_drift"
	}
	if hasResource {
		return "resource_drift"
	}
	return "general_drift"
}

func (s *Service) shouldRollback(impactScore float64, driftDetected bool, drift DriftDetails) bool {
	// Auto-rollback criteria:
	// 1. Impact score is very negative (< -0.7) - significant degradation
	// 2. Critical drift detected
	// 3. Multiple high-severity drifts

	if impactScore < -0.7 {
		return true
	}

	if driftDetected {
		if drift.Severity == "critical" {
			return true
		}
		if drift.Severity == "high" && impactScore < -0.4 {
			return true
		}
	}

	return false
}
