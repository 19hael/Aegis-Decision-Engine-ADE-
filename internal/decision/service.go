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
	policyEngine   *policy.Engine
	decisionStore  *postgres.DecisionStore
	logger         *slog.Logger
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
	
	// Generate IDs
	decisionID := fmt.Sprintf("dec-%d", time.Now().UnixNano())
	traceID := fmt.Sprintf("trace-%d", time.Now().UnixNano())

	// Evaluate policy
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
			SnapshotID:      req.Features.ServiceID + "-snap", // Simplified
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
