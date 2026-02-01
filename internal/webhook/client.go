package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/aegis-decision-engine/ade/internal/circuitbreaker"
)

// Client sends webhooks with retries and circuit breaker
type Client struct {
	httpClient       *http.Client
	logger           *slog.Logger
	maxRetries       int
	baseBackoff      time.Duration
	maxBackoff       time.Duration
	circuitBreaker   *circuitbreaker.CircuitBreaker
}

// Config holds webhook client configuration
type Config struct {
	Timeout              time.Duration
	MaxRetries           int
	BaseBackoff          time.Duration
	MaxBackoff           time.Duration
	EnableCircuitBreaker bool
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Timeout:              30 * time.Second,
		MaxRetries:           3,
		BaseBackoff:          time.Second,
		MaxBackoff:           30 * time.Second,
		EnableCircuitBreaker: true,
	}
}

// NewClient creates a new webhook client
func NewClient(config Config, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}

	client := &Client{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger:      logger,
		maxRetries:  config.MaxRetries,
		baseBackoff: config.BaseBackoff,
		maxBackoff:  config.MaxBackoff,
	}

	if config.EnableCircuitBreaker {
		client.circuitBreaker = circuitbreaker.New("webhook", circuitbreaker.DefaultConfig())
	}

	return client
}

// Request represents a webhook request
type Request struct {
	URL     string                 `json:"url"`
	Method  string                 `json:"method"`
	Headers map[string]string      `json:"headers"`
	Payload map[string]interface{} `json:"payload"`
	ID      string                 `json:"id"`
}

// Response represents a webhook response
type Response struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
	Duration   time.Duration     `json:"duration_ms"`
	Attempts   int               `json:"attempts"`
	Error      string            `json:"error,omitempty"`
}

// Send sends a webhook with retries
func (c *Client) Send(ctx context.Context, req *Request) (*Response, error) {
	if req.Method == "" {
		req.Method = http.MethodPost
	}

	if _, err := url.Parse(req.URL); err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	response := &Response{
		Headers: make(map[string]string),
	}

	if c.circuitBreaker != nil {
		err := c.circuitBreaker.Execute(ctx, func() error {
			resp, err := c.sendWithRetries(ctx, req)
			if resp != nil {
				*response = *resp
			}
			return err
		})
		return response, err
	}

	return c.sendWithRetries(ctx, req)
}

func (c *Client) sendWithRetries(ctx context.Context, req *Request) (*Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		resp, err := c.doRequest(ctx, req)

		if err == nil && resp.StatusCode < 500 {
			resp.Attempts = attempt + 1
			return resp, nil
		}

		lastErr = err
		if resp != nil {
			lastErr = fmt.Errorf("status %d: %s", resp.StatusCode, string(resp.Body))
		}

		if attempt < c.maxRetries {
			backoff := c.calculateBackoff(attempt)
			c.logger.Warn("webhook failed, retrying",
				"attempt", attempt+1,
				"max_retries", c.maxRetries,
				"backoff", backoff,
				"error", lastErr,
			)
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("webhook failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

func (c *Client) doRequest(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()

	body, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Webhook-ID", req.ID)
	httpReq.Header.Set("X-Webhook-Attempt", "1")

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	respBody, _ := io.ReadAll(httpResp.Body)

	resp := &Response{
		StatusCode: httpResp.StatusCode,
		Body:       respBody,
		Duration:   time.Since(start),
	}

	for k, v := range httpResp.Header {
		if len(v) > 0 {
			resp.Headers[k] = v[0]
		}
	}

	return resp, nil
}

func (c *Client) calculateBackoff(attempt int) time.Duration {
	backoff := c.baseBackoff * time.Duration(1<<uint(attempt))
	if backoff > c.maxBackoff {
		backoff = c.maxBackoff
	}
	jitter := time.Duration(float64(backoff) * 0.1)
	return backoff + jitter
}

// GetCircuitBreakerStats returns circuit breaker statistics
func (c *Client) GetCircuitBreakerStats() map[string]interface{} {
	if c.circuitBreaker == nil {
		return map[string]interface{}{"enabled": false}
	}
	return c.circuitBreaker.Stats()
}
