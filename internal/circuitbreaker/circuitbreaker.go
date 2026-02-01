package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota   // Normal operation
	StateOpen                  // Failing, reject requests
	StateHalfOpen              // Testing if service recovered
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config holds circuit breaker configuration
type Config struct {
	MaxFailures      int           // Max failures before opening
	ResetTimeout     time.Duration // Time before trying again
	HalfOpenMaxCalls int           // Max calls in half-open state
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		MaxFailures:      5,
		ResetTimeout:     30 * time.Second,
		HalfOpenMaxCalls: 3,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config      Config
	mu          sync.RWMutex
	state       State
	failures    int
	lastFailure time.Time
	successes   int // For half-open state
	name        string
}

// New creates a new circuit breaker
func New(name string, config Config) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
		name:   name,
	}
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if err := cb.canExecute(); err != nil {
		return err
	}

	err := fn()
	cb.recordResult(err)
	return err
}

// ExecuteWithFallback runs fn with fallback on failure
func (cb *CircuitBreaker) ExecuteWithFallback(ctx context.Context, fn func() error, fallback func() error) error {
	if err := cb.canExecute(); err != nil {
		if fallback != nil {
			return fallback()
		}
		return err
	}

	err := fn()
	cb.recordResult(err)
	
	if err != nil && fallback != nil {
		return fallback()
	}
	return err
}

func (cb *CircuitBreaker) canExecute() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil
	case StateOpen:
		if time.Since(cb.lastFailure) > cb.config.ResetTimeout {
			cb.state = StateHalfOpen
			cb.successes = 0
			return nil
		}
		return fmt.Errorf("circuit breaker OPEN for %s", cb.name)
	case StateHalfOpen:
		if cb.successes >= cb.config.HalfOpenMaxCalls {
			return fmt.Errorf("circuit breaker HALF-OPEN: max calls reached")
		}
		return nil
	}
	return nil
}

func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err == nil {
		switch cb.state {
		case StateClosed:
			cb.failures = 0
		case StateHalfOpen:
			cb.successes++
			if cb.successes >= cb.config.HalfOpenMaxCalls {
				cb.state = StateClosed
				cb.failures = 0
				cb.successes = 0
			}
		}
	} else {
		switch cb.state {
		case StateClosed:
			cb.failures++
			cb.lastFailure = time.Now()
			if cb.failures >= cb.config.MaxFailures {
				cb.state = StateOpen
			}
		case StateHalfOpen:
			cb.state = StateOpen
			cb.lastFailure = time.Now()
		}
	}
}

// State returns current state
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns current statistics
func (cb *CircuitBreaker) Stats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return map[string]interface{}{
		"name":         cb.name,
		"state":        cb.state.String(),
		"failures":     cb.failures,
		"successes":    cb.successes,
		"last_failure": cb.lastFailure,
	}
}

// Reset forces the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
}

// ErrCircuitOpen is returned when circuit is open
var ErrCircuitOpen = errors.New("circuit breaker is open")
