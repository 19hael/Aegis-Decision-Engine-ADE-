package validator

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/aegis-decision-engine/ade/internal/models"
	"github.com/xeipuuv/gojsonschema"
)

// EventValidator validates events against JSON Schema
type EventValidator struct {
	schemas map[models.EventType]*gojsonschema.Schema
}

// NewEventValidator creates a new validator with schemas
func NewEventValidator() (*EventValidator, error) {
	v := &EventValidator{
		schemas: make(map[models.EventType]*gojsonschema.Schema),
	}

	// Define schemas for each event type
	metricsSchema := `{
		"type": "object",
		"required": ["cpu", "latency_ms", "error_rate", "requests_per_second"],
		"properties": {
			"cpu": {"type": "number", "minimum": 0, "maximum": 100},
			"latency_ms": {"type": "number", "minimum": 0},
			"error_rate": {"type": "number", "minimum": 0, "maximum": 1},
			"requests_per_second": {"type": "number", "minimum": 0},
			"queue_depth": {"type": "integer", "minimum": 0}
		}
	}`

	alertSchema := `{
		"type": "object",
		"required": ["alert_type", "severity"],
		"properties": {
			"alert_type": {"type": "string"},
			"severity": {"type": "string", "enum": ["low", "medium", "high", "critical"]},
			"message": {"type": "string"}
		}
	}`

	customSchema := `{
		"type": "object",
		"required": ["event_name"],
		"properties": {
			"event_name": {"type": "string", "minLength": 1},
			"payload": {"type": "object"}
		}
	}`

	// Compile schemas
	schemas := map[models.EventType]string{
		models.EventTypeMetrics: metricsSchema,
		models.EventTypeAlert:   alertSchema,
		models.EventTypeCustom:  customSchema,
	}

	for eventType, schemaJSON := range schemas {
		schema, err := gojsonschema.NewSchema(gojsonschema.NewStringLoader(schemaJSON))
		if err != nil {
			return nil, fmt.Errorf("failed to compile schema for %s: %w", eventType, err)
		}
		v.schemas[eventType] = schema
	}

	return v, nil
}

// ValidateEvent validates an event payload against its schema
func (v *EventValidator) ValidateEvent(eventType models.EventType, payload json.RawMessage) error {
	schema, ok := v.schemas[eventType]
	if !ok {
		return fmt.Errorf("no schema defined for event type: %s", eventType)
	}

	result, err := schema.Validate(gojsonschema.NewBytesLoader(payload))
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if !result.Valid() {
		errors := make([]string, 0, len(result.Errors()))
		for _, err := range result.Errors() {
			errors = append(errors, err.String())
		}
		return fmt.Errorf("validation failed: %v", errors)
	}

	return nil
}

// ValidateServiceID validates service ID format
func ValidateServiceID(serviceID string) error {
	if serviceID == "" {
		return fmt.Errorf("service_id is required")
	}
	if len(serviceID) > 255 {
		return fmt.Errorf("service_id too long (max 255 chars)")
	}
	// Allow alphanumeric, hyphens, underscores, dots
	valid := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`).MatchString(serviceID)
	if !valid {
		return fmt.Errorf("service_id contains invalid characters")
	}
	return nil
}

// ValidateEventID validates event ID format
func ValidateEventID(eventID string) error {
	if eventID == "" {
		return fmt.Errorf("event_id is required")
	}
	if len(eventID) > 255 {
		return fmt.Errorf("event_id too long")
	}
	return nil
}
