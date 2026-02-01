-- Migration 000001: Create base tables for ADE
-- Created: 2026-02-01

-- Extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Events table: event-sourcing light, todo evento entrante se persiste
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id VARCHAR(255) UNIQUE NOT NULL,
    idempotency_key VARCHAR(255) UNIQUE NOT NULL,
    service_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_event_type CHECK (event_type IN ('metrics', 'alert', 'custom'))
);

CREATE INDEX idx_events_service_id ON events(service_id);
CREATE INDEX idx_events_timestamp ON events(timestamp);
CREATE INDEX idx_events_event_type ON events(event_type);
CREATE INDEX idx_events_idempotency ON events(idempotency_key);

-- Feature snapshots: estado calculado por servicio
CREATE TABLE feature_snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    snapshot_id VARCHAR(255) UNIQUE NOT NULL,
    service_id VARCHAR(255) NOT NULL,
    features JSONB NOT NULL,
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until TIMESTAMPTZ,
    event_ids JSONB NOT NULL DEFAULT '[]', -- array de event_ids usados
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_valid_until CHECK (valid_until > calculated_at)
);

CREATE INDEX idx_feature_snapshots_service_id ON feature_snapshots(service_id);
CREATE INDEX idx_feature_snapshots_calculated_at ON feature_snapshots(calculated_at);
CREATE INDEX idx_feature_snapshots_valid_until ON feature_snapshots(valid_until);

-- Policies: definiciones de reglas versionadas
CREATE TABLE policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    dsl_yaml TEXT NOT NULL,
    effective_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    effective_until TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(policy_id, version),
    CONSTRAINT chk_effective_dates CHECK (effective_until IS NULL OR effective_until > effective_from)
);

CREATE INDEX idx_policies_policy_id ON policies(policy_id);
CREATE INDEX idx_policies_active ON policies(is_active, effective_from, effective_until);

-- Decision records: registro de cada decisión tomada
CREATE TABLE decision_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    decision_id VARCHAR(255) UNIQUE NOT NULL,
    idempotency_key VARCHAR(255) UNIQUE NOT NULL,
    service_id VARCHAR(255) NOT NULL,
    policy_id VARCHAR(255) NOT NULL,
    policy_version VARCHAR(50) NOT NULL,
    snapshot_id VARCHAR(255) NOT NULL,
    decision_type VARCHAR(100) NOT NULL,
    decision_result VARCHAR(50) NOT NULL, -- allow, deny, simulate, etc
    actions JSONB NOT NULL DEFAULT '[]',
    confidence_score DECIMAL(5,4) CHECK (confidence_score >= 0 AND confidence_score <= 1),
    simulation_run_id VARCHAR(255),
    dry_run BOOLEAN NOT NULL DEFAULT FALSE,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_decision_type CHECK (decision_type IN ('autoscale', 'ratelimit', 'circuitbreaker', 'custom')),
    CONSTRAINT chk_decision_result CHECK (decision_result IN ('allow', 'deny', 'throttle', 'simulate', 'error'))
);

CREATE INDEX idx_decisions_service_id ON decision_records(service_id);
CREATE INDEX idx_decisions_executed_at ON decision_records(executed_at);
CREATE INDEX idx_decisions_policy ON decision_records(policy_id, policy_version);
CREATE INDEX idx_decisions_simulation ON decision_records(simulation_run_id);

-- Decision traces: explicación estructurada de por qué se tomó la decisión
CREATE TABLE decision_traces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trace_id VARCHAR(255) UNIQUE NOT NULL,
    decision_id VARCHAR(255) NOT NULL REFERENCES decision_records(decision_id) ON DELETE CASCADE,
    policy_id VARCHAR(255) NOT NULL,
    policy_version VARCHAR(50) NOT NULL,
    trace_data JSONB NOT NULL, -- estructura detallada de reglas evaluadas
    rules_evaluated JSONB NOT NULL DEFAULT '[]',
    rules_matched JSONB NOT NULL DEFAULT '[]',
    features_used JSONB NOT NULL DEFAULT '{}',
    execution_time_ms INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_traces_decision_id ON decision_traces(decision_id);
CREATE INDEX idx_traces_policy ON decision_traces(policy_id, policy_version);

-- Simulation runs: ejecuciones de simulación Monte Carlo
CREATE TABLE simulation_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id VARCHAR(255) UNIQUE NOT NULL,
    service_id VARCHAR(255) NOT NULL,
    policy_id VARCHAR(255) NOT NULL,
    policy_version VARCHAR(50) NOT NULL,
    snapshot_id VARCHAR(255) NOT NULL,
    scenario_name VARCHAR(255) NOT NULL,
    horizon_minutes INTEGER NOT NULL CHECK (horizon_minutes > 0),
    iterations INTEGER NOT NULL CHECK (iterations > 0),
    results JSONB NOT NULL, -- resultados de la simulación
    cost_projection DECIMAL(15,4),
    risk_score DECIMAL(5,4) CHECK (risk_score >= 0 AND risk_score <= 1),
    recommendation VARCHAR(50),
    status VARCHAR(50) NOT NULL DEFAULT 'running',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_status CHECK (status IN ('running', 'completed', 'failed', 'timeout')),
    CONSTRAINT chk_horizon CHECK (horizon_minutes BETWEEN 5 AND 15)
);

CREATE INDEX idx_simulations_service_id ON simulation_runs(service_id);
CREATE INDEX idx_simulations_status ON simulation_runs(status);
CREATE INDEX idx_simulations_snapshot ON simulation_runs(snapshot_id);

-- Action records: acciones ejecutadas o programadas
CREATE TABLE action_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    action_id VARCHAR(255) UNIQUE NOT NULL,
    decision_id VARCHAR(255) NOT NULL REFERENCES decision_records(decision_id) ON DELETE CASCADE,
    action_type VARCHAR(100) NOT NULL,
    action_payload JSONB NOT NULL,
    target_service VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    dry_run BOOLEAN NOT NULL DEFAULT FALSE,
    scheduled_at TIMESTAMPTZ,
    executed_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    webhook_url VARCHAR(500),
    webhook_response JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_action_type CHECK (action_type IN ('scale_up', 'scale_down', 'throttle', 'unthrottle', 'open_circuit', 'close_circuit', 'webhook')),
    CONSTRAINT chk_action_status CHECK (status IN ('pending', 'scheduled', 'executing', 'completed', 'failed', 'cancelled'))
);

CREATE INDEX idx_actions_decision_id ON action_records(decision_id);
CREATE INDEX idx_actions_status ON action_records(status);
CREATE INDEX idx_actions_service ON action_records(target_service);
CREATE INDEX idx_actions_scheduled ON action_records(scheduled_at) WHERE scheduled_at IS NOT NULL;

-- Feedback records: medición de impacto y drift detection
CREATE TABLE feedback_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    feedback_id VARCHAR(255) UNIQUE NOT NULL,
    action_id VARCHAR(255) NOT NULL REFERENCES action_records(action_id) ON DELETE CASCADE,
    decision_id VARCHAR(255) NOT NULL,
    service_id VARCHAR(255) NOT NULL,
    feedback_type VARCHAR(100) NOT NULL,
    metrics_before JSONB NOT NULL DEFAULT '{}',
    metrics_after JSONB NOT NULL DEFAULT '{}',
    impact_score DECIMAL(5,4), -- -1.0 a 1.0 (negativo = degradación)
    drift_detected BOOLEAN NOT NULL DEFAULT FALSE,
    drift_details JSONB,
    rollback_recommended BOOLEAN NOT NULL DEFAULT FALSE,
    rollback_executed BOOLEAN NOT NULL DEFAULT FALSE,
    observation_window_minutes INTEGER NOT NULL DEFAULT 5,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_feedback_type CHECK (feedback_type IN ('immediate', 'delayed', 'scheduled'))
);

CREATE INDEX idx_feedback_action_id ON feedback_records(action_id);
CREATE INDEX idx_feedback_decision_id ON feedback_records(decision_id);
CREATE INDEX idx_feedback_service_id ON feedback_records(service_id);
CREATE INDEX idx_feedback_drift ON feedback_records(drift_detected, rollback_recommended);

-- Trigger para updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_policies_updated_at BEFORE UPDATE ON policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_action_records_updated_at BEFORE UPDATE ON action_records
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comentarios de documentación
COMMENT ON TABLE events IS 'Todos los eventos entrantes, event-sourcing light';
COMMENT ON TABLE feature_snapshots IS 'Snapshots de features calculados por servicio';
COMMENT ON TABLE policies IS 'Definiciones de políticas versionadas en DSL YAML';
COMMENT ON TABLE decision_records IS 'Registro inmutable de decisiones tomadas';
COMMENT ON TABLE decision_traces IS 'Trazas de auditoría de decisiones';
COMMENT ON TABLE simulation_runs IS 'Ejecuciones de simulación Monte Carlo';
COMMENT ON TABLE action_records IS 'Acciones ejecutadas o programadas';
COMMENT ON TABLE feedback_records IS 'Feedback y detección de drift post-acción';
