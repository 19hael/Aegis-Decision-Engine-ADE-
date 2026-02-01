package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventValidate(t *testing.T) {
	tests := []struct {
		name    string
		event   Event
		wantErr error
	}{
		{
			name: "valid event",
			event: Event{
				EventID:        "evt-001",
				IdempotencyKey: "idemp-001",
				ServiceID:      "test-service",
				EventType:      EventTypeMetrics,
				Payload:        json.RawMessage(`{"cpu": 50}`),
				Timestamp:      time.Now(),
			},
			wantErr: nil,
		},
		{
			name: "missing event id",
			event: Event{
				IdempotencyKey: "idemp-001",
				ServiceID:      "test-service",
				EventType:      EventTypeMetrics,
				Payload:        json.RawMessage(`{}`),
			},
			wantErr: ErrInvalidEventID,
		},
		{
			name: "missing idempotency key",
			event: Event{
				EventID:   "evt-001",
				ServiceID: "test-service",
				EventType: EventTypeMetrics,
				Payload:   json.RawMessage(`{}`),
			},
			wantErr: ErrMissingIdempotencyKey,
		},
		{
			name: "missing service id",
			event: Event{
				EventID:        "evt-001",
				IdempotencyKey: "idemp-001",
				EventType:      EventTypeMetrics,
				Payload:        json.RawMessage(`{}`),
			},
			wantErr: ErrMissingServiceID,
		},
		{
			name: "missing payload",
			event: Event{
				EventID:        "evt-001",
				IdempotencyKey: "idemp-001",
				ServiceID:      "test-service",
				EventType:      EventTypeMetrics,
			},
			wantErr: ErrMissingPayload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestEventGetMetricsPayload(t *testing.T) {
	t.Run("valid metrics payload", func(t *testing.T) {
		event := Event{
			EventType: EventTypeMetrics,
			Payload: json.RawMessage(`{
				"cpu": 75.5,
				"latency_ms": 450,
				"error_rate": 0.05,
				"requests_per_second": 1000
			}`),
		}

		metrics, err := event.GetMetricsPayload()
		require.NoError(t, err)
		assert.Equal(t, 75.5, metrics.CPU)
		assert.Equal(t, 450.0, metrics.Latency)
		assert.Equal(t, 0.05, metrics.ErrorRate)
		assert.Equal(t, 1000.0, metrics.RequestsPerSec)
	})

	t.Run("not metrics event", func(t *testing.T) {
		event := Event{
			EventType: EventTypeAlert,
			Payload:   json.RawMessage(`{}`),
		}

		_, err := event.GetMetricsPayload()
		assert.ErrorIs(t, err, ErrNotMetricsEvent)
	})

	t.Run("invalid json", func(t *testing.T) {
		event := Event{
			EventType: EventTypeMetrics,
			Payload:   json.RawMessage(`{invalid`),
		}

		_, err := event.GetMetricsPayload()
		assert.Error(t, err)
	})
}

func TestCalculateFeatures(t *testing.T) {
	t.Run("empty events", func(t *testing.T) {
		_, err := CalculateFeatures("test", []Event{})
		assert.Error(t, err)
	})

	t.Run("calculates features from metrics", func(t *testing.T) {
		events := []Event{
			{
				EventType: EventTypeMetrics,
				Payload:   mustMarshal(MetricsPayload{CPU: 80, Latency: 100, ErrorRate: 0.05, RequestsPerSec: 1000, QueueDepth: 10}),
			},
			{
				EventType: EventTypeMetrics,
				Payload:   mustMarshal(MetricsPayload{CPU: 70, Latency: 120, ErrorRate: 0.03, RequestsPerSec: 900, QueueDepth: 8}),
			},
		}

		features, err := CalculateFeatures("test-service", events)
		require.NoError(t, err)
		assert.Equal(t, "test-service", features.ServiceID)
		assert.True(t, features.CPUCurrent > 0)
		assert.True(t, features.HealthScore >= 0 && features.HealthScore <= 1)
	})
}

func mustMarshal(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
