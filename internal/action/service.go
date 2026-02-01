package action

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aegis-decision-engine/ade/internal/models"
	"github.com/aegis-decision-engine/ade/internal/webhook"
)

// Service handles action execution
type Service struct {
	webhookClient *webhook.Client
	webhookURL    string
	logger        *slog.Logger
	dryRun        bool
}

// NewService creates a new action service
func NewService(webhookURL string, dryRun bool, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	
	webhookConfig := webhook.DefaultConfig()
	webhookConfig.EnableCircuitBreaker = true
	
	return &Service{
		webhookClient: webhook.NewClient(webhookConfig, logger),
		webhookURL:    webhookURL,
		logger:        logger,
		dryRun:        dryRun,
	}
}

// ActionRequest represents a request to execute an action
type ActionRequest struct {
	ActionID       string              `json:"action_id"`
	DecisionID     string              `json:"decision_id"`
	ActionType     models.ActionType   `json:"action_type"`
	TargetService  string              `json:"target_service"`
	Payload        map[string]interface{} `json:"payload"`
	DryRun         bool                `json:"dry_run"`
	ScheduledAt    *time.Time          `json:"scheduled_at,omitempty"`
	WebhookURL     string              `json:"webhook_url,omitempty"`
}

// Validate validates the action request
func (r *ActionRequest) Validate() error {
	if r.ActionID == "" {
		return fmt.Errorf("action_id is required")
	}
	if r.DecisionID == "" {
		return fmt.Errorf("decision_id is required")
	}
	if r.ActionType == "" {
		return fmt.Errorf("action_type is required")
	}
	if r.TargetService == "" {
		return fmt.Errorf("target_service is required")
	}
	return nil
}

// ActionResult represents the result of executing an action
type ActionResult struct {
	ActionID      string                 `json:"action_id"`
	Status        string                 `json:"status"`
	DryRun        bool                   `json:"dry_run"`
	ExecutedAt    time.Time              `json:"executed_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	WebhookURL    string                 `json:"webhook_url"`
	ResponseCode  int                    `json:"response_code,omitempty"`
	ResponseBody  string                 `json:"response_body,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Execute runs an action
func (s *Service) Execute(ctx context.Context, req *ActionRequest) (*ActionResult, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	result := &ActionResult{
		ActionID:   req.ActionID,
		Status:     "executing",
		DryRun:     req.DryRun || s.dryRun,
		ExecutedAt: time.Now(),
		WebhookURL: req.WebhookURL,
	}

	if result.DryRun {
		result.Status = "dry_run"
		result.Metadata = map[string]interface{}{
			"action_type":    req.ActionType,
			"target_service": req.TargetService,
			"payload":        req.Payload,
			"message":        "action would have been executed",
		}
		s.logger.Info("action dry run",
			"action_id", req.ActionID,
			"type", req.ActionType,
			"target", req.TargetService,
		)
		return result, nil
	}

	// Build webhook payload
	webhookPayload := map[string]interface{}{
		"action_id":      req.ActionID,
		"decision_id":    req.DecisionID,
		"action_type":    req.ActionType,
		"target_service": req.TargetService,
		"payload":        req.Payload,
		"timestamp":      time.Now(),
	}

	// Send webhook
	webhookURL := req.WebhookURL
	if webhookURL == "" {
		webhookURL = s.webhookURL
	}

	if webhookURL != "" {
		webhookReq := &webhook.Request{
			ID:      req.ActionID,
			URL:     webhookURL,
			Method:  "POST",
			Payload: webhookPayload,
			Headers: map[string]string{
				"X-Action-Type": string(req.ActionType),
				"X-Service-ID":  req.TargetService,
			},
		}
		
		webhookResp, err := s.webhookClient.Send(ctx, webhookReq)
		if err != nil {
			result.Status = "failed"
			result.ErrorMessage = err.Error()
			s.logger.Error("webhook failed",
				"action_id", req.ActionID,
				"error", err,
			)
			return result, err
		}
		
		result.ResponseCode = webhookResp.StatusCode
		result.ResponseBody = string(webhookResp.Body)
	}

	now := time.Now()
	result.Status = "completed"
	result.CompletedAt = &now

	s.logger.Info("action executed",
		"action_id", req.ActionID,
		"type", req.ActionType,
		"target", req.TargetService,
		"duration_ms", time.Since(result.ExecutedAt).Milliseconds(),
	)

	return result, nil
}

// Schedule schedules an action for future execution
func (s *Service) Schedule(ctx context.Context, req *ActionRequest) (*ActionResult, error) {
	if req.ScheduledAt == nil {
		return nil, fmt.Errorf("scheduled_at is required for scheduling")
	}

	result := &ActionResult{
		ActionID:   req.ActionID,
		Status:     "scheduled",
		DryRun:     req.DryRun,
		ExecutedAt: time.Now(),
		Metadata: map[string]interface{}{
			"action_type":    req.ActionType,
			"target_service": req.TargetService,
			"payload":        req.Payload,
			"scheduled_for":  req.ScheduledAt,
		},
	}

	s.logger.Info("action scheduled",
		"action_id", req.ActionID,
		"type", req.ActionType,
		"scheduled_for", req.ScheduledAt,
	)

	return result, nil
}

// ExecuteBatch executes multiple actions
func (s *Service) ExecuteBatch(ctx context.Context, requests []*ActionRequest) ([]*ActionResult, error) {
	results := make([]*ActionResult, 0, len(requests))

	for _, req := range requests {
		result, err := s.Execute(ctx, req)
		if err != nil {
			s.logger.Error("batch action failed", "action_id", req.ActionID, "error", err)
			result = &ActionResult{
				ActionID:     req.ActionID,
				Status:       "failed",
				ErrorMessage: err.Error(),
				ExecutedAt:   time.Now(),
			}
		}
		results = append(results, result)
	}

	return results, nil
}



// GetActionTypeFromString converts string to ActionType
func GetActionTypeFromString(s string) models.ActionType {
	switch s {
	case "scale_up":
		return models.ActionTypeScaleUp
	case "scale_down":
		return models.ActionTypeScaleDown
	case "throttle":
		return models.ActionTypeThrottle
	case "unthrottle":
		return models.ActionTypeUnthrottle
	case "open_circuit":
		return models.ActionTypeOpenCircuit
	case "close_circuit":
		return models.ActionTypeCloseCircuit
	default:
		return models.ActionTypeWebhook
	}
}
