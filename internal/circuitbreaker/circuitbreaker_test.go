package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreakerStateTransitions(t *testing.T) {
	config := Config{
		MaxFailures:      3,
		ResetTimeout:     100 * time.Millisecond,
		HalfOpenMaxCalls: 2,
	}
	
	cb := New("test", config)
	
	// Initial state should be Closed
	assert.Equal(t, StateClosed, cb.State())
	
	// Execute successful calls
	for i := 0; i < 3; i++ {
		err := cb.Execute(context.Background(), func() error {
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, StateClosed, cb.State())
	}
}

func TestCircuitBreakerOpensAfterFailures(t *testing.T) {
	config := Config{
		MaxFailures:      3,
		ResetTimeout:     1 * time.Second,
		HalfOpenMaxCalls: 2,
	}
	
	cb := New("test", config)
	testErr := errors.New("test error")
	
	// Generate failures to open circuit
	for i := 0; i < 3; i++ {
		err := cb.Execute(context.Background(), func() error {
			return testErr
		})
		assert.Error(t, err)
	}
	
	// Circuit should be open now
	assert.Equal(t, StateOpen, cb.State())
	
	// Subsequent calls should fail immediately
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker OPEN")
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	config := Config{
		MaxFailures:      2,
		ResetTimeout:     50 * time.Millisecond,
		HalfOpenMaxCalls: 2,
	}
	
	cb := New("test", config)
	testErr := errors.New("test error")
	
	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func() error { return testErr })
	}
	assert.Equal(t, StateOpen, cb.State())
	
	// Wait for reset timeout
	time.Sleep(100 * time.Millisecond)
	
	// Next call should transition to half-open
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateHalfOpen, cb.State())
}

func TestCircuitBreakerClosesAfterSuccess(t *testing.T) {
	config := Config{
		MaxFailures:      2,
		ResetTimeout:     50 * time.Millisecond,
		HalfOpenMaxCalls: 2,
	}
	
	cb := New("test", config)
	testErr := errors.New("test error")
	
	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func() error { return testErr })
	}
	
	// Wait and transition to half-open
	time.Sleep(100 * time.Millisecond)
	
	// Success calls in half-open should close the circuit
	for i := 0; i < 2; i++ {
		err := cb.Execute(context.Background(), func() error {
			return nil
		})
		assert.NoError(t, err)
	}
	
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreakerReset(t *testing.T) {
	config := Config{
		MaxFailures:      2,
		ResetTimeout:     1 * time.Hour,
		HalfOpenMaxCalls: 2,
	}
	
	cb := New("test", config)
	testErr := errors.New("test error")
	
	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func() error { return testErr })
	}
	assert.Equal(t, StateOpen, cb.State())
	
	// Reset manually
	cb.Reset()
	assert.Equal(t, StateClosed, cb.State())
	
	// Should work normally again
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestCircuitBreakerExecuteWithFallback(t *testing.T) {
	config := Config{
		MaxFailures:      1,
		ResetTimeout:     1 * time.Hour,
		HalfOpenMaxCalls: 1,
	}
	
	cb := New("test", config)
	testErr := errors.New("test error")
	fallbackCalled := false
	
	// First failure - should not call fallback
	cb.Execute(context.Background(), func() error { return testErr })
	
	// Circuit open - should call fallback
	err := cb.ExecuteWithFallback(context.Background(), 
		func() error { return testErr },
		func() error { 
			fallbackCalled = true
			return nil 
		},
	)
	
	assert.NoError(t, err)
	assert.True(t, fallbackCalled)
}

func TestCircuitBreakerStats(t *testing.T) {
	config := Config{
		MaxFailures:      3,
		ResetTimeout:     1 * time.Second,
		HalfOpenMaxCalls: 2,
	}
	
	cb := New("test-cb", config)
	
	stats := cb.Stats()
	assert.Equal(t, "test-cb", stats["name"])
	assert.Equal(t, "closed", stats["state"])
}
