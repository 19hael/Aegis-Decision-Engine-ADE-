package simulation

import (
	"encoding/json"
	"net/http"

	"github.com/aegis-decision-engine/ade/internal/models"
)

// Handler handles HTTP requests for simulations
type Handler struct {
	service *Service
}

// NewHandler creates a new simulation handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the simulation routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/simulations", h.handleListSimulations)
	mux.HandleFunc("/simulations/run", h.handleRunSimulation)
	mux.HandleFunc("/simulations/{id}", h.handleGetSimulation)
}

func (h *Handler) handleRunSimulation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req SimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	result, err := h.service.Run(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "simulation failed: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) handleListSimulations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// TODO: implement with query params
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"simulations": []interface{}{},
		"count":       0,
	})
}

func (h *Handler) handleGetSimulation(w http.ResponseWriter, r *http.Request) {
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

// QuickSimRequest helper to create a quick simulation request
func QuickSimRequest(serviceID string, features *models.ServiceFeatures) *SimulationRequest {
	return &SimulationRequest{
		ServiceID:      serviceID,
		PolicyID:       "autoscale_policy",
		PolicyVersion:  "1.0",
		SnapshotID:     serviceID + "-current",
		Scenario:       "normal",
		HorizonMinutes: 10,
		Iterations:     1000,
		CurrentState:   features,
	}
}
