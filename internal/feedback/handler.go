package feedback

import (
	"encoding/json"
	"net/http"
)

// Handler handles HTTP requests for feedback
type Handler struct {
	service *Service
}

// NewHandler creates a new feedback handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the feedback routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/feedback", h.handleRecordFeedback)
	mux.HandleFunc("/feedback/{id}", h.handleGetFeedback)
	mux.HandleFunc("/rollback", h.handleRollback)
	mux.HandleFunc("/services/{id}/drift", h.handleCheckDrift)
}

func (h *Handler) handleRecordFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req FeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	result, err := h.service.RecordFeedback(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "feedback recording failed: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) handleRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req RollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	result, err := h.service.Rollback(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "rollback failed: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) handleCheckDrift(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	serviceID := r.PathValue("id")
	if serviceID == "" {
		writeError(w, http.StatusBadRequest, "service id required")
		return
	}

	// TODO: implement actual drift check against current metrics
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service_id":    serviceID,
		"drift_detected": false,
		"message":       "drift check not yet implemented",
	})
}

func (h *Handler) handleGetFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// TODO: implement
	writeError(w, http.StatusNotImplemented, "not implemented")
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// QuickFeedback helper to create a feedback request
func QuickFeedback(actionID, decisionID, serviceID string, before, after map[string]float64) *FeedbackRequest {
	return &FeedbackRequest{
		ActionID:              actionID,
		DecisionID:            decisionID,
		ServiceID:             serviceID,
		FeedbackType:          "immediate",
		MetricsBefore:         before,
		MetricsAfter:          after,
		ObservationWindowMins: 5,
	}
}
