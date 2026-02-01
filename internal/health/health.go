package health

import (
	"context"
	"sync"
	"time"
)

// Status represents the health status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// Check represents a single health check
type Check struct {
	Name         string        `json:"name"`
	Status       Status        `json:"status"`
	ResponseTime time.Duration `json:"response_time_ms"`
	Error        string        `json:"error,omitempty"`
	LastChecked  time.Time     `json:"last_checked"`
}

// Result represents the overall health result
type Result struct {
	Status    Status    `json:"status"`
	Checks    []Check   `json:"checks"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
}

// Checker performs health checks
type Checker struct {
	mu      sync.RWMutex
	checks  map[string]CheckFunc
	version string
}

// CheckFunc is a function that performs a health check
type CheckFunc func(ctx context.Context) error

// NewChecker creates a new health checker
func NewChecker(version string) *Checker {
	return &Checker{
		checks:  make(map[string]CheckFunc),
		version: version,
	}
}

// Register adds a health check
func (c *Checker) Register(name string, fn CheckFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks[name] = fn
}

// Check runs all health checks
func (c *Checker) Check(ctx context.Context) *Result {
	c.mu.RLock()
	checks := make(map[string]CheckFunc, len(c.checks))
	for k, v := range c.checks {
		checks[k] = v
	}
	c.mu.RUnlock()

	result := &Result{
		Status:    StatusHealthy,
		Checks:    make([]Check, 0, len(checks)),
		Version:   c.version,
		Timestamp: time.Now(),
	}

	for name, fn := range checks {
		check := c.runCheck(ctx, name, fn)
		result.Checks = append(result.Checks, check)

		if check.Status == StatusUnhealthy {
			result.Status = StatusUnhealthy
		}
	}

	return result
}

func (c *Checker) runCheck(ctx context.Context, name string, fn CheckFunc) Check {
	start := time.Now()
	check := Check{
		Name:        name,
		Status:      StatusHealthy,
		LastChecked: time.Now(),
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := fn(ctx)
	check.ResponseTime = time.Since(start)

	if err != nil {
		check.Status = StatusUnhealthy
		check.Error = err.Error()
	}

	return check
}
