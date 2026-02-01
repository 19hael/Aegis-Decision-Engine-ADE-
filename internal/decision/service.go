package decision

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/aegis-decision-engine/ade/internal/models"
	"github.com/aegis-decision-engine/ade/internal/policy"
	"github.com/aegis-decision-engine/ade/internal/storage/postgres"
)

// Service handles decision making
type Service struct {
	policyEngine  *policy.Engine
	decisionStore *postgres.DecisionStore
	logger        *slog.Logger
}

// NewService creates a new decision service
func NewService(policyEngine *policy.Engine, decisionStore *postgres.DecisionStore, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		policyEngine:  policyEngine,
		decisionStore: decisionStore,
		logger:        logger,
	}
}

// MakeDecision creates a decision based on features and policy
func (s *Service) MakeDecision(ctx context.Context, req *models.DecisionRequest, pol *policy.Policy) (*models.DecisionResponse, error) {
	start := time.Now()
	
	decisionID := fmt.Sprintf("dec-%d", time.Now().UnixNano())
	traceID := fmt.Sprintf("trace-%d", time.Now().UnixNano())

	result, allResults := s.policyEngine.Evaluate(ctx, pol, req.Features)

	// Build actions
	actions := []models.Action{}
	if result.Matched && result.Action != "" {
		actionPayload, _ := json.Marshal(result.ActionPayload)
		actions = append(actions, models.Action{
			Type:    result.Action,
			Payload: actionPayload,
			Target:  req.ServiceID,
			Cost:    getActionCost(pol, result.RuleID),
			Risk:    getActionRisk(pol, result.RuleID),
		})
	}

	// Determine result
	decisionResult := models.DecisionResultAllow
	if result.Matched {
		switch result.Action {
		case models.ActionTypeScaleUp, models.ActionTypeScaleDown:
			decisionResult = models.DecisionResultAllow
		case models.ActionTypeThrottle:
			decisionResult = models.DecisionResultThrottle
		case models.ActionTypeOpenCircuit:
			decisionResult = models.DecisionResultDeny
		}
	}

	executionTimeMs := int(time.Since(start).Milliseconds())

	// Store decision record
	if s.decisionStore != nil {
		actionsJSON, _ := json.Marshal(actions)
		confidence := 0.8
		if result.Confidence > 0 {
			confidence = result.Confidence
		}

		decisionRecord := &models.DecisionRecord{
			DecisionID:      decisionID,
			IdempotencyKey:  req.IdempotencyKey,
			ServiceID:       req.ServiceID,
			PolicyID:        pol.ID,
			PolicyVersion:   pol.Version,
			SnapshotID:      req.Features.ServiceID + "-snap",
			DecisionType:    models.DecisionType(pol.Type),
			DecisionResult:  decisionResult,
			Actions:         actionsJSON,
			ConfidenceScore: &confidence,
			DryRun:          req.DryRun,
			ExecutedAt:      time.Now(),
		}

		if err := s.decisionStore.Store(ctx, decisionRecord); err != nil {
			s.logger.Warn("failed to store decision", "error", err)
		}

		// Store trace
		trace := &models.DecisionTrace{
			TraceID:         traceID,
			DecisionID:      decisionID,
			PolicyID:        pol.ID,
			PolicyVersion:   pol.Version,
			TraceData:       mustMarshal(result),
			RulesEvaluated:  mustMarshal(allResults),
			RulesMatched:    mustMarshal([]string{result.RuleID}),
			FeaturesUsed:    mustMarshal(req.Features),
			ExecutionTimeMs: executionTimeMs,
		}

		if err := s.decisionStore.StoreTrace(ctx, trace); err != nil {
			s.logger.Warn("failed to store trace", "error", err)
		}
	}

	s.logger.Info("decision made",
		"decision_id", decisionID,
		"service_id", req.ServiceID,
		"result", decisionResult,
		"matched", result.Matched,
		"action", result.Action,
		"duration_ms", executionTimeMs,
	)

	return &models.DecisionResponse{
		DecisionID:     decisionID,
		DecisionResult: decisionResult,
		Actions:        actions,
		Confidence:     result.Confidence,
		TraceID:        traceID,
		DryRun:         req.DryRun,
		Timestamp:      time.Now(),
	}, nil
}

// GetDecision retrieves a decision by ID
func (s *Service) GetDecision(ctx context.Context, decisionID string) (*models.DecisionRecord, error) {
	if s.decisionStore == nil {
		return nil, fmt.Errorf("decision store not available")
	}
	return s.decisionStore.GetByID(ctx, decisionID)
}

// GetDecisionTrace retrieves the trace for a decision
func (s *Service) GetDecisionTrace(ctx context.Context, decisionID string) (*models.DecisionTrace, error) {
	if s.decisionStore == nil {
		return nil, fmt.Errorf("decision store not available")
	}
	return s.decisionStore.GetTraceByDecisionID(ctx, decisionID)
}

// ListDecisions lists decisions with filters
func (s *Service) ListDecisions(ctx context.Context, filters models.DecisionFilters) ([]*models.DecisionRecord, error) {
	if s.decisionStore == nil {
		return nil, fmt.Errorf("decision store not available")
	}
	return s.decisionStore.ListByFilters(ctx, filters)
}

func getActionCost(pol *policy.Policy, ruleID string) float64 {
	for _, r := range pol.Rules {
		if r.ID == ruleID {
			return r.Action.Cost
		}
	}
	return 0
}

func getActionRisk(pol *policy.Policy, ruleID string) float64 {
	for _, r := range pol.Rules {
		if r.ID == ruleID {
			return r.Action.Risk
		}
	}
	return 0
}

func mustMarshal(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
