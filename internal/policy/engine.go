package policy

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/aegis-decision-engine/ade/internal/models"
)

// Engine evaluates policies against features
type Engine struct {
	logger *slog.Logger
}

// NewEngine creates a new policy engine
func NewEngine(logger *slog.Logger) *Engine {
	if logger == nil {
		logger = slog.Default()
	}
	return &Engine{logger: logger}
}

// EvaluationResult represents the result of policy evaluation
type EvaluationResult struct {
	Matched       bool              `json:"matched"`
	RuleID        string            `json:"rule_id,omitempty"`
	Action        models.ActionType `json:"action,omitempty"`
	ActionPayload map[string]interface{} `json:"action_payload,omitempty"`
	Reason        string            `json:"reason,omitempty"`
	Confidence    float64           `json:"confidence"`
	EvaluatedAt   time.Time         `json:"evaluated_at"`
}

// Evaluate evaluates a policy against service features
func (e *Engine) Evaluate(ctx context.Context, policy *Policy, features *models.ServiceFeatures) (*EvaluationResult, []EvaluationResult) {
	start := time.Now()
	allResults := make([]EvaluationResult, 0, len(policy.Rules))

	// Sort rules by priority (highest first) using bubble sort for simplicity
	sortedRules := make([]Rule, len(policy.Rules))
	copy(sortedRules, policy.Rules)
	for i := 0; i < len(sortedRules); i++ {
		for j := i + 1; j < len(sortedRules); j++ {
			if sortedRules[i].Priority < sortedRules[j].Priority {
				sortedRules[i], sortedRules[j] = sortedRules[j], sortedRules[i]
			}
		}
	}

	// Evaluate each rule
	for _, rule := range sortedRules {
		result := e.evaluateRule(&rule, features)
		result.EvaluatedAt = time.Now()
		allResults = append(allResults, result)

		if result.Matched {
			e.logger.Info("rule matched",
				"policy_id", policy.ID,
				"policy_version", policy.Version,
				"rule_id", rule.ID,
				"action", rule.Action.Type,
				"duration_ms", time.Since(start).Milliseconds(),
			)
			return &result, allResults
		}
	}

	// No rules matched - return default
	return &EvaluationResult{
		Matched:     false,
		Reason:      "no rules matched",
		Confidence:  1.0,
		EvaluatedAt: time.Now(),
	}, allResults
}

func (e *Engine) evaluateRule(rule *Rule, features *models.ServiceFeatures) EvaluationResult {
	matched := e.evaluateCondition(&rule.When, features)

	if !matched {
		return EvaluationResult{
			Matched:    false,
			RuleID:     rule.ID,
			Confidence: 1.0,
		}
	}

	actionType := models.ActionType(rule.Action.Type)

	return EvaluationResult{
		Matched:       true,
		RuleID:        rule.ID,
		Action:        actionType,
		ActionPayload: rule.Action.Params,
		Reason:        fmt.Sprintf("condition matched for rule %s", rule.ID),
		Confidence:    calculateConfidence(rule, features),
	}
}

func (e *Engine) evaluateCondition(cond *Condition, features *models.ServiceFeatures) bool {
	// Handle compound conditions
	if len(cond.All) > 0 {
		for _, c := range cond.All {
			if !e.evaluateCondition(&c, features) {
				return false
			}
		}
		return true
	}

	if len(cond.Any) > 0 {
		for _, c := range cond.Any {
			if e.evaluateCondition(&c, features) {
				return true
			}
		}
		return false
	}

	if cond.Not != nil {
		return !e.evaluateCondition(cond.Not, features)
	}

	// Simple condition evaluation
	if cond.Fact == "" || cond.Op == "" {
		return true
	}

	factValue := getFactValue(cond.Fact, features)
	if factValue == nil {
		return false
	}

	return compare(factValue, cond.Op, cond.Value)
}

func getFactValue(fact string, features *models.ServiceFeatures) interface{} {
	v := reflect.ValueOf(features).Elem()
	field := v.FieldByName(fact)
	
	if !field.IsValid() {
		switch fact {
		case "cpu":
			field = v.FieldByName("CPUCurrent")
		case "latency":
			field = v.FieldByName("LatencyP95")
		case "error_rate":
			field = v.FieldByName("ErrorRate")
		case "rps":
			field = v.FieldByName("RequestsPerSec")
		case "queue_depth":
			field = v.FieldByName("QueueDepth")
		case "health_score":
			field = v.FieldByName("HealthScore")
		case "load_score":
			field = v.FieldByName("LoadScore")
		}
	}

	if !field.IsValid() {
		return nil
	}

	return field.Interface()
}

func compare(factValue interface{}, op string, targetValue interface{}) bool {
	factNum := toFloat64(factValue)
	targetNum := toFloat64(targetValue)

	if factNum == nil || targetNum == nil {
		factStr := fmt.Sprintf("%v", factValue)
		targetStr := fmt.Sprintf("%v", targetValue)
		switch op {
		case "==":
			return factStr == targetStr
		case "!=":
			return factStr != targetStr
		default:
			return false
		}
	}

	f := *factNum
	t := *targetNum

	switch op {
	case ">":
		return f > t
	case ">=":
		return f >= t
	case "<":
		return f < t
	case "<=":
		return f <= t
	case "==":
		return f == t
	case "!=":
		return f != t
	default:
		return false
	}
}

func toFloat64(v interface{}) *float64 {
	switch val := v.(type) {
	case float64:
		return &val
	case float32:
		f := float64(val)
		return &f
	case int:
		f := float64(val)
		return &f
	case int32:
		f := float64(val)
		return &f
	case int64:
		f := float64(val)
		return &f
	default:
		return nil
	}
}

func calculateConfidence(rule *Rule, features *models.ServiceFeatures) float64 {
	baseConfidence := 0.8

	if rule.Priority > 50 {
		baseConfidence += 0.1
	}

	if features.HealthScore < 0.3 {
		baseConfidence -= 0.15
	}

	if baseConfidence > 1.0 {
		return 1.0
	}
	if baseConfidence < 0.0 {
		return 0.0
	}
	return baseConfidence
}
