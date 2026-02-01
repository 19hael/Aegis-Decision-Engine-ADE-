package policy

import (
	"context"
	"fmt"
	"testing"

	"github.com/aegis-decision-engine/ade/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPolicy(t *testing.T) {
	// Create a test policy file
	policy := &Policy{
		ID:      "test_policy",
		Version: "1.0",
		Name:    "Test Policy",
		Type:    "autoscale",
		Rules: []Rule{
			{
				ID:       "high_cpu",
				Name:     "High CPU Rule",
				Priority: 100,
				When: Condition{
					Fact:  "CPUCurrent",
					Op:    ">=",
					Value: 80.0,
				},
				Action: Action{
					Type: "scale_up",
					Cost: 10.0,
					Risk: 0.1,
				},
			},
		},
	}

	err := policy.Validate()
	require.NoError(t, err)
	assert.Equal(t, "test_policy", policy.ID)
	assert.Equal(t, 1, len(policy.Rules))
}

func TestPolicyValidation(t *testing.T) {
	tests := []struct {
		name    string
		policy  Policy
		wantErr bool
	}{
		{
			name: "valid policy",
			policy: Policy{
				ID:      "valid",
				Version: "1.0",
				Rules:   []Rule{{ID: "r1", Name: "Rule 1", Action: Action{Type: "scale_up"}}},
			},
			wantErr: false,
		},
		{
			name: "missing id",
			policy: Policy{
				Version: "1.0",
				Rules:   []Rule{{ID: "r1", Name: "Rule 1", Action: Action{Type: "scale_up"}}},
			},
			wantErr: true,
		},
		{
			name: "missing version",
			policy: Policy{
				ID:    "test",
				Rules: []Rule{{ID: "r1", Name: "Rule 1", Action: Action{Type: "scale_up"}}},
			},
			wantErr: true,
		},
		{
			name: "no rules",
			policy: Policy{
				ID:      "test",
				Version: "1.0",
				Rules:   []Rule{},
			},
			wantErr: true,
		},
		{
			name: "duplicate rule id",
			policy: Policy{
				ID:      "test",
				Version: "1.0",
				Rules: []Rule{
					{ID: "r1", Name: "Rule 1", Action: Action{Type: "scale_up"}},
					{ID: "r1", Name: "Rule 2", Action: Action{Type: "scale_down"}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEngineEvaluate(t *testing.T) {
	engine := NewEngine(nil)

	policy := &Policy{
		ID:      "test",
		Version: "1.0",
		Rules: []Rule{
			{
				ID:       "high_cpu",
				Name:     "High CPU",
				Priority: 100,
				When: Condition{
					Fact:  "CPUCurrent",
					Op:    ">=",
					Value: 80.0,
				},
				Action: Action{
					Type: "scale_up",
				},
			},
			{
				ID:       "low_cpu",
				Name:     "Low CPU",
				Priority: 50,
				When: Condition{
					Fact:  "CPUCurrent",
					Op:    "<=",
					Value: 30.0,
				},
				Action: Action{
					Type: "scale_down",
				},
			},
		},
	}

	t.Run("matches high cpu rule", func(t *testing.T) {
		features := &models.ServiceFeatures{
			CPUCurrent: 85.0,
		}

		result, _ := engine.Evaluate(context.Background(), policy, features)

		assert.True(t, result.Matched)
		assert.Equal(t, "high_cpu", result.RuleID)
		assert.Equal(t, models.ActionTypeScaleUp, result.Action)
	})

	t.Run("matches low cpu rule", func(t *testing.T) {
		features := &models.ServiceFeatures{
			CPUCurrent: 25.0,
		}

		result, _ := engine.Evaluate(context.Background(), policy, features)

		assert.True(t, result.Matched)
		assert.Equal(t, "low_cpu", result.RuleID)
		assert.Equal(t, models.ActionTypeScaleDown, result.Action)
	})

	t.Run("no rule matches", func(t *testing.T) {
		features := &models.ServiceFeatures{
			CPUCurrent: 50.0,
		}

		result, _ := engine.Evaluate(context.Background(), policy, features)

		assert.False(t, result.Matched)
	})
}

func TestCompoundConditions(t *testing.T) {
	engine := NewEngine(nil)

	tests := []struct {
		name     string
		cond     Condition
		features *models.ServiceFeatures
		want     bool
	}{
		{
			name: "all conditions true",
			cond: Condition{
				All: []Condition{
					{Fact: "CPUCurrent", Op: ">=", Value: 70.0},
					{Fact: "LatencyP95", Op: ">=", Value: 500.0},
				},
			},
			features: &models.ServiceFeatures{
				CPUCurrent: 80.0,
				LatencyP95: 600.0,
			},
			want: true,
		},
		{
			name: "all conditions false (one fails)",
			cond: Condition{
				All: []Condition{
					{Fact: "CPUCurrent", Op: ">=", Value: 70.0},
					{Fact: "LatencyP95", Op: ">=", Value: 500.0},
				},
			},
			features: &models.ServiceFeatures{
				CPUCurrent: 80.0,
				LatencyP95: 400.0,
			},
			want: false,
		},
		{
			name: "any condition true",
			cond: Condition{
				Any: []Condition{
					{Fact: "CPUCurrent", Op: ">=", Value: 90.0},
					{Fact: "ErrorRate", Op: ">=", Value: 0.5},
				},
			},
			features: &models.ServiceFeatures{
				CPUCurrent: 95.0,
				ErrorRate:  0.1,
			},
			want: true,
		},
		{
			name: "not condition",
			cond: Condition{
				Not: &Condition{
					Fact:  "HealthScore",
					Op:    ">=",
					Value: 0.8,
				},
			},
			features: &models.ServiceFeatures{
				HealthScore: 0.5,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.evaluateCondition(&tt.cond, tt.features)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		factValue    interface{}
		op           string
		targetValue  interface{}
		want         bool
	}{
		{80.0, ">=", 70.0, true},
		{50.0, ">=", 70.0, false},
		{30.0, "<=", 50.0, true},
		{80.0, "<", 90.0, true},
		{80.0, ">", 70.0, true},
		{80.0, "==", 80.0, true},
		{80.0, "!=", 70.0, true},
		{"active", "==", "active", true},
		{"active", "!=", "inactive", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v %s %v", tt.factValue, tt.op, tt.targetValue), func(t *testing.T) {
			got := compare(tt.factValue, tt.op, tt.targetValue)
			assert.Equal(t, tt.want, got)
		})
	}
}
