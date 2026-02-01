package ingest

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aegis-decision-engine/ade/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request IngestRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: IngestRequest{
				EventID:        "evt-001",
				IdempotencyKey: "idemp-001",
				ServiceID:      "test-service",
				EventType:      models.EventTypeMetrics,
				Payload:        json.RawMessage(`{"cpu": 50}`),
				Timestamp:      time.Now(),
			},
			wantErr: false,
		},
		{
			name: "missing event id",
			request: IngestRequest{
				IdempotencyKey: "idemp-001",
				ServiceID:      "test-service",
				EventType:      models.EventTypeMetrics,
				Payload:        json.RawMessage(`{}`),
			},
			wantErr: true,
		},
		{
			name: "missing service id",
			request: IngestRequest{
				EventID:        "evt-001",
				IdempotencyKey: "idemp-001",
				EventType:      models.EventTypeMetrics,
				Payload:        json.RawMessage(`{}`),
			},
			wantErr: true,
		},
		{
			name: "missing payload",
			request: IngestRequest{
				EventID:        "evt-001",
				IdempotencyKey: "idemp-001",
				ServiceID:      "test-service",
				EventType:      models.EventTypeMetrics,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMetricsRequestHelper(t *testing.T) {
	req := MetricsRequest("test-service", 75.5, 450, 0.05, 1000, 10)

	require.NotNil(t, req)
	assert.Equal(t, "test-service", req.ServiceID)
	assert.Equal(t, models.EventTypeMetrics, req.EventType)
	assert.NotEmpty(t, req.EventID)
	assert.NotEmpty(t, req.IdempotencyKey)

	var payload models.MetricsPayload
	err := json.Unmarshal(req.Payload, &payload)
	require.NoError(t, err)
	assert.Equal(t, 75.5, payload.CPU)
	assert.Equal(t, 450.0, payload.Latency)
	assert.Equal(t, 0.05, payload.ErrorRate)
	assert.Equal(t, 1000.0, payload.RequestsPerSec)
	assert.Equal(t, 10, payload.QueueDepth)
}
