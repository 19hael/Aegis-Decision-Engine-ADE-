package models

import (
	"fmt"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) ValidationError {
	return ValidationError{Field: field, Message: message}
}

// Common errors
var (
	// Event errors
	ErrEventNotFound        = fmt.Errorf("event not found")
	ErrDuplicateEvent       = fmt.Errorf("duplicate event")
	ErrInvalidEventType     = fmt.Errorf("invalid event type")
	ErrInvalidEventID       = NewValidationError("event_id", "event ID is required")
	ErrMissingIdempotencyKey = NewValidationError("idempotency_key", "idempotency key is required")
	ErrMissingServiceID     = NewValidationError("service_id", "service ID is required")
	ErrMissingPayload       = NewValidationError("payload", "payload is required")
	ErrNotMetricsEvent      = fmt.Errorf("event type is not metrics")
	
	// Decision errors
	ErrDecisionNotFound   = fmt.Errorf("decision not found")
	ErrPolicyNotFound     = fmt.Errorf("policy not found")
	ErrInvalidPolicy      = fmt.Errorf("invalid policy")
	ErrPolicyEvalFailed   = fmt.Errorf("policy evaluation failed")
	
	// Feature errors
	ErrFeatureNotFound    = fmt.Errorf("feature snapshot not found")
	ErrNoMetricsEvents    = fmt.Errorf("no metrics events found")
	
	// Action errors
	ErrActionNotFound     = fmt.Errorf("action not found")
	ErrActionFailed       = fmt.Errorf("action execution failed")
	ErrActionTimeout      = fmt.Errorf("action execution timeout")
	
	// Storage errors
	ErrNotFound           = fmt.Errorf("record not found")
	ErrDuplicateKey       = fmt.Errorf("duplicate key")
	ErrConnection         = fmt.Errorf("database connection error")
	ErrTransaction        = fmt.Errorf("transaction error")
	
	// General errors
	ErrUnauthorized       = fmt.Errorf("unauthorized")
	ErrForbidden          = fmt.Errorf("forbidden")
	ErrInvalidInput       = fmt.Errorf("invalid input")
	ErrInternal           = fmt.Errorf("internal error")
	ErrNotImplemented     = fmt.Errorf("not implemented")
)

// IsNotFound checks if error is a not found error
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return err == ErrNotFound || err == ErrEventNotFound || 
		err == ErrDecisionNotFound || err == ErrFeatureNotFound ||
		err == ErrPolicyNotFound || err == ErrActionNotFound
}

// IsValidationError checks if error is a validation error
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(ValidationError)
	return ok
}
