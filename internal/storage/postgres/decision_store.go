package postgres

import (
	"context"
	"fmt"

	"github.com/aegis-decision-engine/ade/internal/models"
	"github.com/jackc/pgx/v5"
)

// DecisionStore handles decision persistence
type DecisionStore struct {
	client *Client
}

// NewDecisionStore creates a new decision store
func NewDecisionStore(client *Client) *DecisionStore {
	return &DecisionStore{client: client}
}

// Store persists a decision record
func (s *DecisionStore) Store(ctx context.Context, decision *models.DecisionRecord) error {
	query := `
		INSERT INTO decision_records (
			decision_id, idempotency_key, service_id, policy_id, policy_version,
			snapshot_id, decision_type, decision_result, actions, 
			confidence_score, simulation_run_id, dry_run, executed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id, created_at`

	err := s.client.Pool().QueryRow(ctx, query,
		decision.DecisionID,
		decision.IdempotencyKey,
		decision.ServiceID,
		decision.PolicyID,
		decision.PolicyVersion,
		decision.SnapshotID,
		decision.DecisionType,
		decision.DecisionResult,
		decision.Actions,
		decision.ConfidenceScore,
		decision.SimulationRunID,
		decision.DryRun,
		decision.ExecutedAt,
	).Scan(&decision.ID, &decision.CreatedAt)

	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to store decision: %w", err)
	}

	return nil
}

// GetByID retrieves a decision by its ID
func (s *DecisionStore) GetByID(ctx context.Context, decisionID string) (*models.DecisionRecord, error) {
	query := `
		SELECT id, decision_id, idempotency_key, service_id, policy_id, policy_version,
			snapshot_id, decision_type, decision_result, actions, 
			confidence_score, simulation_run_id, dry_run, executed_at, created_at
		FROM decision_records WHERE decision_id = $1`

	var decision models.DecisionRecord
	err := s.client.Pool().QueryRow(ctx, query, decisionID).Scan(
		&decision.ID, &decision.DecisionID, &decision.IdempotencyKey, &decision.ServiceID,
		&decision.PolicyID, &decision.PolicyVersion, &decision.SnapshotID,
		&decision.DecisionType, &decision.DecisionResult, &decision.Actions,
		&decision.ConfidenceScore, &decision.SimulationRunID, &decision.DryRun,
		&decision.ExecutedAt, &decision.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get decision: %w", err)
	}

	return &decision, nil
}

// StoreTrace persists a decision trace
func (s *DecisionStore) StoreTrace(ctx context.Context, trace *models.DecisionTrace) error {
	query := `
		INSERT INTO decision_traces (
			trace_id, decision_id, policy_id, policy_version, trace_data,
			rules_evaluated, rules_matched, features_used, execution_time_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`

	err := s.client.Pool().QueryRow(ctx, query,
		trace.TraceID,
		trace.DecisionID,
		trace.PolicyID,
		trace.PolicyVersion,
		trace.TraceData,
		trace.RulesEvaluated,
		trace.RulesMatched,
		trace.FeaturesUsed,
		trace.ExecutionTimeMs,
	).Scan(&trace.ID, &trace.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to store trace: %w", err)
	}

	return nil
}

// GetTraceByDecisionID retrieves the trace for a decision
func (s *DecisionStore) GetTraceByDecisionID(ctx context.Context, decisionID string) (*models.DecisionTrace, error) {
	query := `
		SELECT id, trace_id, decision_id, policy_id, policy_version, trace_data,
			rules_evaluated, rules_matched, features_used, execution_time_ms, created_at
		FROM decision_traces WHERE decision_id = $1`

	var trace models.DecisionTrace
	err := s.client.Pool().QueryRow(ctx, query, decisionID).Scan(
		&trace.ID, &trace.TraceID, &trace.DecisionID, &trace.PolicyID,
		&trace.PolicyVersion, &trace.TraceData, &trace.RulesEvaluated,
		&trace.RulesMatched, &trace.FeaturesUsed, &trace.ExecutionTimeMs, &trace.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get trace: %w", err)
	}

	return &trace, nil
}
