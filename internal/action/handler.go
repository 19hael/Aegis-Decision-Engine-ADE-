package action

import (
	"encoding/json"
	"net/http"

	"github.com/aegis-decision-engine/ade/internal/models"
)

// Handler handles HTTP requests for actions
type Handler struct {
	service *Service
}

// NewHandler creates a new action handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the action routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/actions", h.handleListActions)
	mux.HandleFunc("/actions/execute", h.handleExecuteAction)
	mux.HandleFunc("/actions/schedule", h.handleScheduleAction)
	mux.HandleFunc("/actions/batch", h.handleBatchExecute)
	mux.HandleFunc("/actions/{id}", h.handleGetAction)
}

func (h *Handler) handleExecuteAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req ActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	result, err := h.service.Execute(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "execution failed: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if result.DryRun {
		w.WriteHeader(http.StatusAccepted)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) handleScheduleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req ActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	result, err := h.service.Schedule(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "scheduling failed: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) handleBatchExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var requests []*ActionRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	results, err := h.service.ExecuteBatch(r.Context(), requests)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "batch execution failed: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
		"count":   len(results),
	})
}

func (h *Handler) handleListActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// TODO: implement with query params
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"actions": []interface{}{},
		"count":   0,
	})
}

func (h *Handler) handleGetAction(w http.ResponseWriter, r *http.Request) {
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

// CreateActionRequest helper to create an action request
func CreateActionRequest(decisionID, actionType, serviceID string, dryRun bool) *ActionRequest {
	payload := map[string]interface{}{
		"timestamp": "now",
	}

	// Add specific params based on action type
	switch models.ActionType(actionType) {
	case models.ActionTypeScaleUp:
		payload["instances"] = 2
		payload["urgency"] = "high"
	case models.ActionTypeScaleDown:
		payload["instances"] = 1
		payload["min_instances"] = 2
	case models.ActionTypeOpenCircuit:
		payload["duration"] = "60s"
	}

	return &ActionRequest{
		ActionID:      "act-" + decisionID[:8],
		DecisionID:    decisionID,
		ActionType:    models.ActionType(actionType),
		TargetService: serviceID,
		Payload:       payload,
		DryRun:        dryRun,
	}
}
