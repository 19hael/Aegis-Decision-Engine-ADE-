package simulation

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"time"

	"github.com/aegis-decision-engine/ade/internal/models"
)

// Service runs Monte Carlo simulations
type Service struct {
	logger *slog.Logger
}

// NewService creates a new simulation service
func NewService(logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	rand.Seed(time.Now().UnixNano())
	return &Service{
		logger: logger,
	}
}

// SimulationRequest represents a request to run a simulation
type SimulationRequest struct {
	ServiceID      string                  `json:"service_id"`
	PolicyID       string                  `json:"policy_id"`
	PolicyVersion  string                  `json:"policy_version"`
	SnapshotID     string                  `json:"snapshot_id"`
	Scenario       string                  `json:"scenario"`
	HorizonMinutes int                     `json:"horizon_minutes"`
	Iterations     int                     `json:"iterations"`
	CurrentState   *models.ServiceFeatures `json:"current_state"`
}

// Validate validates the simulation request
func (r *SimulationRequest) Validate() error {
	if r.ServiceID == "" {
		return fmt.Errorf("service_id is required")
	}
	if r.CurrentState == nil {
		return fmt.Errorf("current_state is required")
	}
	if r.HorizonMinutes < 5 || r.HorizonMinutes > 15 {
		r.HorizonMinutes = 10
	}
	if r.Iterations < 100 {
		r.Iterations = 1000
	}
	if r.Scenario == "" {
		r.Scenario = "normal"
	}
	return nil
}

// SimulationResult represents the result of a simulation
type SimulationResult struct {
	RunID           string               `json:"run_id"`
	Status          string               `json:"status"`
	Scenario        string               `json:"scenario"`
	HorizonMinutes  int                  `json:"horizon_minutes"`
	Iterations      int                  `json:"iterations"`
	ProjectedStates []ProjectedState     `json:"projected_states"`
	Aggregates      SimulationAggregates `json:"aggregates"`
	CostProjection  float64              `json:"cost_projection"`
	RiskScore       float64              `json:"risk_score"`
	Recommendation  string               `json:"recommendation"`
	Confidence      float64              `json:"confidence"`
	StartedAt       time.Time            `json:"started_at"`
	CompletedAt     time.Time            `json:"completed_at"`
}

// ProjectedState represents a state at a future point in time
type ProjectedState struct {
	Minute     int     `json:"minute"`
	CPUAvg     float64 `json:"cpu_avg"`
	CPUP50     float64 `json:"cpu_p50"`
	CPUP95     float64 `json:"cpu_p95"`
	LatencyAvg float64 `json:"latency_avg"`
	ErrorRate  float64 `json:"error_rate"`
}

// SimulationAggregates contains aggregate statistics
type SimulationAggregates struct {
	ProbabilityOverload     float64 `json:"probability_overload"`
	ProbabilityHighLatency  float64 `json:"probability_high_latency"`
	ProbabilityErrorSpike   float64 `json:"probability_error_spike"`
	ExpectedCost            float64 `json:"expected_cost"`
	WorstCaseCost           float64 `json:"worst_case_cost"`
	BestCaseCost            float64 `json:"best_case_cost"`
}

// Run executes a Monte Carlo simulation
func (s *Service) Run(ctx context.Context, req *SimulationRequest) (*SimulationResult, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	start := time.Now()
	runID := fmt.Sprintf("sim-%d", start.UnixNano())

	s.logger.Info("starting simulation",
		"run_id", runID,
		"service_id", req.ServiceID,
		"scenario", req.Scenario,
		"horizon", req.HorizonMinutes,
		"iterations", req.Iterations,
	)

	result := &SimulationResult{
		RunID:          runID,
		Status:         "running",
		Scenario:       req.Scenario,
		HorizonMinutes: req.HorizonMinutes,
		Iterations:     req.Iterations,
		StartedAt:      start,
	}

	// Run Monte Carlo iterations
	projections := make([][]ProjectedState, req.Iterations)
	for i := 0; i < req.Iterations; i++ {
		projections[i] = s.projectState(req.CurrentState, req.HorizonMinutes, req.Scenario)
	}

	// Aggregate results
	result.ProjectedStates = s.aggregateProjections(projections, req.HorizonMinutes)
	result.Aggregates = s.calculateAggregates(projections, req.HorizonMinutes)
	result.CostProjection = s.calculateCostProjection(result.Aggregates, req.Scenario)
	result.RiskScore = s.calculateRiskScore(result.Aggregates)
	result.Recommendation = s.generateRecommendation(result)
	result.Confidence = s.calculateConfidence(req.Iterations)
	result.Status = "completed"
	result.CompletedAt = time.Now()

	s.logger.Info("simulation completed",
		"run_id", runID,
		"duration_ms", time.Since(start).Milliseconds(),
		"risk_score", result.RiskScore,
		"recommendation", result.Recommendation,
	)

	return result, nil
}

func (s *Service) projectState(current *models.ServiceFeatures, horizon int, scenario string) []ProjectedState {
	states := make([]ProjectedState, horizon)
	
	cpu := current.CPUCurrent
	latency := current.LatencyP95
	errorRate := current.ErrorRate

	// Scenario parameters
	cpuTrend := 0.0
	errorTrend := 0.0
	noiseFactor := 0.1

	switch scenario {
	case "high_load":
		cpuTrend = 0.03
		noiseFactor = 0.15
	case "failure":
		errorTrend = 0.02
		noiseFactor = 0.2
	case "recovery":
		cpuTrend = -0.02
		noiseFactor = 0.08
	}

	for minute := 1; minute <= horizon; minute++ {
		cpu = cpu * (1 + cpuTrend + noiseFactor*(rand.Float64()-0.5))
		errorRate = math.Min(1.0, errorRate*(1+errorTrend+noiseFactor*(rand.Float64()-0.5)))
		latency = latency * (1 + (cpu-50)/200 + noiseFactor*(rand.Float64()-0.5))

		cpu = math.Max(0, math.Min(100, cpu))
		errorRate = math.Max(0, math.Min(1, errorRate))
		latency = math.Max(0, latency)

		states[minute-1] = ProjectedState{
			Minute:     minute,
			CPUAvg:     cpu,
			CPUP50:     cpu * (0.9 + 0.2*rand.Float64()),
			CPUP95:     cpu * (1.1 + 0.3*rand.Float64()),
			LatencyAvg: latency,
			ErrorRate:  errorRate,
		}
	}

	return states
}

func (s *Service) aggregateProjections(projections [][]ProjectedState, horizon int) []ProjectedState {
	aggregated := make([]ProjectedState, horizon)

	for minute := 0; minute < horizon; minute++ {
		var cpuSum, latencySum, errorSum float64
		n := len(projections)

		for _, proj := range projections {
			if minute < len(proj) {
				cpuSum += proj[minute].CPUAvg
				latencySum += proj[minute].LatencyAvg
				errorSum += proj[minute].ErrorRate
			}
		}

		aggregated[minute] = ProjectedState{
			Minute:     minute + 1,
			CPUAvg:     cpuSum / float64(n),
			LatencyAvg: latencySum / float64(n),
			ErrorRate:  errorSum / float64(n),
		}
	}

	return aggregated
}

func (s *Service) calculateAggregates(projections [][]ProjectedState, horizon int) SimulationAggregates {
	agg := SimulationAggregates{}
	
	overloadCount := 0
	highLatencyCount := 0
	errorSpikeCount := 0
	totalCost := 0.0
	minCost := math.MaxFloat64
	maxCost := 0.0

	for _, proj := range projections {
		projOverload := false
		projHighLatency := false
		projErrorSpike := false
		projCost := 0.0

		for _, state := range proj {
			if state.CPUAvg > 90 {
				projOverload = true
			}
			if state.LatencyAvg > 1000 {
				projHighLatency = true
			}
			if state.ErrorRate > 0.1 {
				projErrorSpike = true
			}
			projCost += 0.1 + state.CPUAvg/100.0*0.5
		}

		if projOverload {
			overloadCount++
		}
		if projHighLatency {
			highLatencyCount++
		}
		if projErrorSpike {
			errorSpikeCount++
		}

		totalCost += projCost
		if projCost < minCost {
			minCost = projCost
		}
		if projCost > maxCost {
			maxCost = projCost
		}
	}

	n := float64(len(projections))
	agg.ProbabilityOverload = float64(overloadCount) / n
	agg.ProbabilityHighLatency = float64(highLatencyCount) / n
	agg.ProbabilityErrorSpike = float64(errorSpikeCount) / n
	agg.ExpectedCost = totalCost / n
	agg.BestCaseCost = minCost
	agg.WorstCaseCost = maxCost

	return agg
}

func (s *Service) calculateCostProjection(agg SimulationAggregates, scenario string) float64 {
	baseCost := agg.ExpectedCost
	switch scenario {
	case "high_load":
		return baseCost * 1.5
	case "failure":
		return baseCost * 2.0
	case "recovery":
		return baseCost * 0.8
	}
	return baseCost
}

func (s *Service) calculateRiskScore(agg SimulationAggregates) float64 {
	risk := agg.ProbabilityOverload*0.4 + 
		agg.ProbabilityHighLatency*0.3 + 
		agg.ProbabilityErrorSpike*0.3
	return math.Min(1.0, risk)
}

func (s *Service) generateRecommendation(result *SimulationResult) string {
	if result.RiskScore > 0.7 {
		return "scale_up_immediate"
	} else if result.RiskScore > 0.5 {
		return "scale_up_prepare"
	} else if result.RiskScore > 0.3 {
		return "monitor_closely"
	}
	return "maintain"
}

func (s *Service) calculateConfidence(iterations int) float64 {
	return math.Min(0.95, 0.5+float64(iterations)/20000.0)
}
