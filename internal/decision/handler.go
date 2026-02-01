package decision

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/aegis-decision-engine/ade/internal/models"
	"github.com/aegis-decision-engine/ade/internal/policy"
)

// Handler handles HTTP requests for decisions
type Handler struct {
	service  *Service
	policies map[string]*policy.Policy
}

// NewHandler creates a new decision handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service:  service,
		policies: make(map[string]*policy.Policy),
	}
}

// LoadDefaultPolicy loads the default autoscale policy
func (h *Handler) LoadDefaultPolicy() error {
	pol, err := policy.LoadPolicy("policies/autoscale_v1.yaml")
	if err != nil {
		return err
	}
	h.policies[pol.ID] = pol
	return nil
}

// RegisterRoutes registers the decision routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/decisions", h.handleListDecisions)
	mux.HandleFunc("/decisions/{id}", h.handleGetDecision)
	mux.HandleFunc("/decisions/{id}/trace", h.handleGetDecisionTrace)
	mux.HandleFunc("/evaluate", h.handleEvaluate)
	mux.HandleFunc("/policies/load", h.handleLoadPolicy)
}

func (h *Handler) handleEvaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		ServiceID      string                 `json:"service_id"`
		PolicyID       string                 `json:"policy_id"`
		Features       *models.ServiceFeatures `json:"features"`
		DryRun         bool                   `json:"dry_run"`
		IdempotencyKey string                 `json:"idempotency_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.ServiceID == "" {
		writeError(w, http.StatusBadRequest, "service_id is required")
		return
	}

	if req.PolicyID == "" {
		req.PolicyID = "autoscale_policy"
	}

	if req.IdempotencyKey == "" {
		req.IdempotencyKey = "idemp-" + time.Now().Format("20060102150405")
	}

	// Get policy
	pol, ok := h.policies[req.PolicyID]
	if !ok {
		writeError(w, http.StatusNotFound, "policy not found: "+req.PolicyID)
		return
	}

	decisionReq := &models.DecisionRequest{
		ServiceID:      req.ServiceID,
		DecisionType:   models.DecisionTypeAutoScale,
		Features:       req.Features,
		DryRun:         req.DryRun,
		IdempotencyKey: req.IdempotencyKey,
	}

	resp, err := h.service.MakeDecision(r.Context(), decisionReq, pol)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "decision failed: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleLoadPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Path == "" {
		req.Path = "policies/autoscale_v1.yaml"
	}

	pol, err := policy.LoadPolicy(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to load policy: "+err.Error())
		return
	}

	h.policies[pol.ID] = pol

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "loaded",
		"policy":  pol.ID,
		"version": pol.Version,
	})
}

func (h *Handler) handleListDecisions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// TODO: implement with query params
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"decisions": []interface{}{},
		"count":     0,
	})
}

func (h *Handler) handleGetDecision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// TODO: implement
	writeError(w, http.StatusNotImplemented, "not implemented")
}

func (h *Handler) handleGetDecisionTrace(w http.ResponseWriter, r *http.Request) {
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
