package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActionTypeConstants(t *testing.T) {
	assert.Equal(t, ActionType("scale_up"), ActionTypeScaleUp)
	assert.Equal(t, ActionType("scale_down"), ActionTypeScaleDown)
	assert.Equal(t, ActionType("throttle"), ActionTypeThrottle)
	assert.Equal(t, ActionType("unthrottle"), ActionTypeUnthrottle)
	assert.Equal(t, ActionType("open_circuit"), ActionTypeOpenCircuit)
	assert.Equal(t, ActionType("close_circuit"), ActionTypeCloseCircuit)
	assert.Equal(t, ActionType("webhook"), ActionTypeWebhook)
}

func TestDecisionTypeConstants(t *testing.T) {
	assert.Equal(t, DecisionType("autoscale"), DecisionTypeAutoScale)
	assert.Equal(t, DecisionType("ratelimit"), DecisionTypeRateLimit)
	assert.Equal(t, DecisionType("circuitbreaker"), DecisionTypeCircuitBreaker)
	assert.Equal(t, DecisionType("custom"), DecisionTypeCustom)
}

func TestDecisionResultConstants(t *testing.T) {
	assert.Equal(t, DecisionResult("allow"), DecisionResultAllow)
	assert.Equal(t, DecisionResult("deny"), DecisionResultDeny)
	assert.Equal(t, DecisionResult("throttle"), DecisionResultThrottle)
	assert.Equal(t, DecisionResult("simulate"), DecisionResultSimulate)
	assert.Equal(t, DecisionResult("error"), DecisionResultError)
}
