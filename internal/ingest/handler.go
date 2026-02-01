package ingest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/aegis-decision-engine/ade/internal/models"
)

// Handler handles HTTP requests for ingestion
type Handler struct {
	service *Service
}

// NewHandler creates a new ingest handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the ingest routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/ingest", h.handleIngest)
	mux.HandleFunc("/ingest/batch", h.handleBatchIngest)
}

func (h *Handler) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Set timestamp if not provided
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}

	resp, err := h.service.Ingest(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ingestion failed: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleBatchIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var requests []*IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Set timestamps if not provided
	for _, req := range requests {
		if req.Timestamp.IsZero() {
			req.Timestamp = time.Now()
		}
	}

	responses, err := h.service.IngestBatch(r.Context(), requests)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "batch ingestion failed: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": responses,
		"count":   len(responses),
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// MetricsRequest helper to create a metrics ingest request
func MetricsRequest(serviceID string, cpu, latency, errorRate, rps float64, queueDepth int) *IngestRequest {
	payload, _ := json.Marshal(models.MetricsPayload{
		CPU:            cpu,
		Latency:        latency,
		ErrorRate:      errorRate,
		RequestsPerSec: rps,
		QueueDepth:     queueDepth,
	})

	return &IngestRequest{
		EventID:        "evt-" + time.Now().Format("20060102150405") + "-" + serviceID,
		IdempotencyKey: "idemp-" + time.Now().Format("20060102150405") + "-" + serviceID,
		ServiceID:      serviceID,
		EventType:      models.EventTypeMetrics,
		Payload:        payload,
		Timestamp:      time.Now(),
	}
}
