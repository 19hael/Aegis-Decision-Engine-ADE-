package models

import "errors"

// Common errors
var (
	// Event errors
	ErrInvalidEventID        = errors.New("invalid event id")
	ErrMissingIdempotencyKey = errors.New("missing idempotency key")
	ErrMissingServiceID      = errors.New("missing service id")
	ErrMissingEventType      = errors.New("missing event type")
	ErrMissingPayload        = errors.New("missing payload")
	ErrNotMetricsEvent       = errors.New("not a metrics event")
	
	// Decision errors
	ErrInvalidDecisionID = errors.New("invalid decision id")
	ErrInvalidPolicyID   = errors.New("invalid policy id")
	ErrPolicyNotFound    = errors.New("policy not found")
	
	// Storage errors
	ErrNotFound     = errors.New("record not found")
	ErrDuplicateKey = errors.New("duplicate key")
	ErrConnection   = errors.New("database connection error")
)
