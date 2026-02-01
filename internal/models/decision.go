package models

import (
	"encoding/json"
	"time"
)

// DecisionType represents the type of decision
type DecisionType string

const (
	DecisionTypeAutoScale      DecisionType = "autoscale"
	DecisionTypeRateLimit      DecisionType = "ratelimit"
	DecisionTypeCircuitBreaker DecisionType = "circuitbreaker"
	DecisionTypeCustom         DecisionType = "custom"
)

// DecisionResult represents the result of a decision
type DecisionResult string

const (
	DecisionResultAllow    DecisionResult = "allow"
	DecisionResultDeny     DecisionResult = "deny"
	DecisionResultThrottle DecisionResult = "throttle"
	DecisionResultSimulate DecisionResult = "simulate"
	DecisionResultError    DecisionResult = "error"
)

// ActionType represents the type of action to take
type ActionType string

const (
	ActionTypeScaleUp       ActionType = "scale_up"
	ActionTypeScaleDown     ActionType = "scale_down"
	ActionTypeThrottle      ActionType = "throttle"
	ActionTypeUnthrottle    ActionType = "unthrottle"
	ActionTypeOpenCircuit   ActionType = "open_circuit"
	ActionTypeCloseCircuit  ActionType = "close_circuit"
	ActionTypeWebhook       ActionType = "webhook"
)

// DecisionRecord represents a decision made by the system
type DecisionRecord struct {
	ID              string          `json:"id" db:"id"`
	DecisionID      string          `json:"decision_id" db:"decision_id"`
	IdempotencyKey  string          `json:"idempotency_key" db:"idempotency_key"`
	ServiceID       string          `json:"service_id" db:"service_id"`
	PolicyID        string          `json:"policy_id" db:"policy_id"`
	PolicyVersion   string          `json:"policy_version" db:"policy_version"`
	SnapshotID      string          `json:"snapshot_id" db:"snapshot_id"`
	DecisionType    DecisionType    `json:"decision_type" db:"decision_type"`
	DecisionResult  DecisionResult  `json:"decision_result" db:"decision_result"`
	Actions         json.RawMessage `json:"actions" db:"actions"`
	ConfidenceScore *float64        `json:"confidence_score,omitempty" db:"confidence_score"`
	SimulationRunID *string         `json:"simulation_run_id,omitempty" db:"simulation_run_id"`
	DryRun          bool            `json:"dry_run" db:"dry_run"`
	ExecutedAt      time.Time       `json:"executed_at" db:"executed_at"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
}

// DecisionTrace represents the audit trail of a decision
type DecisionTrace struct {
	ID               string          `json:"id" db:"id"`
	TraceID          string          `json:"trace_id" db:"trace_id"`
	DecisionID       string          `json:"decision_id" db:"decision_id"`
	PolicyID         string          `json:"policy_id" db:"policy_id"`
	PolicyVersion    string          `json:"policy_version" db:"policy_version"`
	TraceData        json.RawMessage `json:"trace_data" db:"trace_data"`
	RulesEvaluated   json.RawMessage `json:"rules_evaluated" db:"rules_evaluated"`
	RulesMatched     json.RawMessage `json:"rules_matched" db:"rules_matched"`
	FeaturesUsed     json.RawMessage `json:"features_used" db:"features_used"`
	ExecutionTimeMs  int             `json:"execution_time_ms" db:"execution_time_ms"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
}

// TraceRule represents a single rule evaluation in a trace
type TraceRule struct {
	RuleID      string          `json:"rule_id"`
	Name        string          `json:"name"`
	Condition   string          `json:"condition"`
	Evaluated   bool            `json:"evaluated"`
	Matched     bool            `json:"matched"`
	Priority    int             `json:"priority"`
	Action      ActionType      `json:"action,omitempty"`
	Reason      string          `json:"reason,omitempty"`
}

// Action represents a single action to be executed
type Action struct {
	Type    ActionType      `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Target  string          `json:"target,omitempty"`
	Cost    float64         `json:"cost,omitempty"`
	Risk    float64         `json:"risk,omitempty"`
}

// DecisionRequest represents a request to make a decision
type DecisionRequest struct {
	ServiceID     string         `json:"service_id"`
	DecisionType  DecisionType   `json:"decision_type"`
	Features      *ServiceFeatures `json:"features,omitempty"`
	DryRun        bool           `json:"dry_run"`
	Simulate      bool           `json:"simulate"`
	IdempotencyKey string        `json:"idempotency_key"`
}

// DecisionResponse represents the response from a decision
type DecisionResponse struct {
	DecisionID     string         `json:"decision_id"`
	DecisionResult DecisionResult `json:"result"`
	Actions        []Action       `json:"actions"`
	Confidence     float64        `json:"confidence"`
	TraceID        string         `json:"trace_id"`
	DryRun         bool           `json:"dry_run"`
	Timestamp      time.Time      `json:"timestamp"`
}
