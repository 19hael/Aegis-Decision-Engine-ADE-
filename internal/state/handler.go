package state

import (
	"encoding/json"
	"net/http"
	"time"
)

// Handler handles HTTP requests for state/features
type Handler struct {
	service *Service
}

// NewHandler creates a new state handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the state routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/services/{id}/state", h.handleGetState)
	mux.HandleFunc("/services/{id}/features/calculate", h.handleCalculateFeatures)
}

func (h *Handler) handleGetState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	serviceID := r.PathValue("id")
	if serviceID == "" {
		writeError(w, http.StatusBadRequest, "service id required")
		return
	}

	features, err := h.service.GetLatestFeatures(r.Context(), serviceID)
	if err != nil {
		writeError(w, http.StatusNotFound, "features not found: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(features)
}

func (h *Handler) handleCalculateFeatures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	serviceID := r.PathValue("id")
	if serviceID == "" {
		writeError(w, http.StatusBadRequest, "service id required")
		return
	}

	// Parse optional window parameter
	window := 5 * time.Minute
	if w := r.URL.Query().Get("window"); w != "" {
		if d, err := time.ParseDuration(w); err == nil {
			window = d
		}
	}

	req := &CalculateFeaturesRequest{
		ServiceID: serviceID,
		Window:    window,
	}

	features, err := h.service.CalculateFeatures(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "calculation failed: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service_id": serviceID,
		"features":   features,
		"calculated_at": time.Now(),
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
