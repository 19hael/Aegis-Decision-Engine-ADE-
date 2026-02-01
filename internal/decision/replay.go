package decision

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aegis-decision-engine/ade/internal/models"
	"github.com/aegis-decision-engine/ade/internal/policy"
	"github.com/aegis-decision-engine/ade/internal/storage/postgres"
)

// ReplayService handles decision replay functionality
type ReplayService struct {
	decisionStore *postgres.DecisionStore
	eventStore    *postgres.EventStore
	policyEngine  *policy.Engine
	logger        interface{}
}

// NewReplayService creates a new replay service
func NewReplayService(decisionStore *postgres.DecisionStore, eventStore *postgres.EventStore, policyEngine *policy.Engine) *ReplayService {
	return &ReplayService{
		decisionStore: decisionStore,
		eventStore:    eventStore,
		policyEngine:  policyEngine,
	}
}

// ReplayRequest represents a replay request
type ReplayRequest struct {
	ServiceID    string    `json:"service_id"`
	From         time.Time `json:"from"`
	To           time.Time `json:"to"`
	PolicyID     string    `json:"policy_id,omitempty"`     // Optional: replay with different policy
	PolicyVersion string   `json:"policy_version,omitempty"` // Optional: specific policy version
}

// ReplayResult represents the result of a replay
type ReplayResult struct {
	OriginalDecisionID string                 `json:"original_decision_id"`
	ReplayDecisionID   string                 `json:"replay_decision_id"`
	ServiceID          string                 `json:"service_id"`
	OriginalResult     models.DecisionResult  `json:"original_result"`
	ReplayResult       models.DecisionResult  `json:"replay_result"`
	Match              bool                   `json:"match"`
	Differences        []string               `json:"differences,omitempty"`
	ReplayedAt         time.Time              `json:"replayed_at"`
}

// ReplayRange replays decisions over a time range
func (s *ReplayService) ReplayRange(ctx context.Context, req *ReplayRequest) ([]*ReplayResult, error) {
	// Get original decisions from the time range
	decisions, err := s.getDecisionsInRange(ctx, req.ServiceID, req.From, req.To)
	if err != nil {
		return nil, fmt.Errorf("failed to get decisions: %w", err)
	}

	results := make([]*ReplayResult, 0, len(decisions))

	for _, decision := range decisions {
		result, err := s.replayDecision(ctx, decision, req.PolicyID, req.PolicyVersion)
		if err != nil {
			continue // Log error but continue with others
		}
		results = append(results, result)
	}

	return results, nil
}

// ReplaySingle replays a single decision
func (s *ReplayService) ReplaySingle(ctx context.Context, decisionID string, policyID, policyVersion string) (*ReplayResult, error) {
	decision, err := s.decisionStore.GetByID(ctx, decisionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get decision: %w", err)
	}

	return s.replayDecision(ctx, decision, policyID, policyVersion)
}

func (s *ReplayService) replayDecision(ctx context.Context, original *models.DecisionRecord, overridePolicyID, overridePolicyVersion string) (*ReplayResult, error) {
	// Get the features snapshot at the time of the original decision
	_, err := s.recreateFeatures(ctx, original.ServiceID, original.ExecutedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to recreate features: %w", err)
	}

	// Determine which policy to use
	_ = original.PolicyID
	_ = original.PolicyVersion
	
	if overridePolicyID != "" {
		_ = overridePolicyID
	}
	if overridePolicyVersion != "" {
		_ = overridePolicyVersion
	}

	// Load the policy (in real implementation, would load from DB)
	// For now, we use the same result for comparison
	
	// Create replay decision ID
	replayDecisionID := fmt.Sprintf("replay-%d", time.Now().UnixNano())

	result := &ReplayResult{
		OriginalDecisionID: original.DecisionID,
		ReplayDecisionID:   replayDecisionID,
		ServiceID:          original.ServiceID,
		OriginalResult:     original.DecisionResult,
		ReplayedAt:         time.Now(),
	}

	// Compare results
	result.Match = (original.DecisionResult == result.ReplayResult)
	
	if !result.Match {
		result.Differences = append(result.Differences, 
			fmt.Sprintf("result changed from %s to %s", original.DecisionResult, result.ReplayResult))
	}

	return result, nil
}

func (s *ReplayService) recreateFeatures(ctx context.Context, serviceID string, at time.Time) (*models.ServiceFeatures, error) {
	// Get events leading up to the decision time
	window := 5 * time.Minute
	from := at.Add(-window)
	
	events, err := s.eventStore.GetByService(ctx, serviceID, from, at, 1000)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return nil, fmt.Errorf("no events found for service %s at %v", serviceID, at)
	}

	features, err := models.CalculateFeatures(serviceID, events)
	if err != nil {
		return nil, err
	}

	return features, nil
}

func (s *ReplayService) getDecisionsInRange(ctx context.Context, serviceID string, from, to time.Time) ([]*models.DecisionRecord, error) {
	// This would query the database for decisions in range
	// For now, return empty (implement based on actual decisionStore)
	return []*models.DecisionRecord{}, nil
}

// CompareDecisions compares two decisions and returns differences
func CompareDecisions(original, replay *models.DecisionRecord) []string {
	diffs := []string{}

	if original.DecisionResult != replay.DecisionResult {
		diffs = append(diffs, fmt.Sprintf("result: %s vs %s", original.DecisionResult, replay.DecisionResult))
	}

	if original.PolicyVersion != replay.PolicyVersion {
		diffs = append(diffs, fmt.Sprintf("policy_version: %s vs %s", original.PolicyVersion, replay.PolicyVersion))
	}

	// Compare actions
	var origActions, replayActions []models.Action
	json.Unmarshal(original.Actions, &origActions)
	json.Unmarshal(replay.Actions, &replayActions)

	if len(origActions) != len(replayActions) {
		diffs = append(diffs, fmt.Sprintf("action_count: %d vs %d", len(origActions), len(replayActions)))
	}

	return diffs
}
