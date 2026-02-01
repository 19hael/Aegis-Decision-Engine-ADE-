<div align="center">

# ⚡ Aegis Decision Engine (ADE)

**Autonomous, Auditable, Simulation-Driven Decisioning for Real-Time Systems**

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16+-336791?logo=postgresql)](https://postgresql.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Build](https://img.shields.io/badge/Build-Passing-brightgreen)]()

[Overview](#overview) • [Quick Start](#quick-start) • [Architecture](#architecture) • [API](#api-reference) • [Demo](#demo)

</div>

---

## Overview

ADE is a production-grade decision engine that automates operational responses to real-time system metrics. It combines **event sourcing**, **Monte Carlo simulation**, and **policy-driven decisioning** to provide autonomous, auditable, and reversible operational decisions.

### Key Capabilities

| Capability | Description |
|------------|-------------|
| 🔄 **Event Ingestion** | Ingest metrics at scale with idempotency guarantees |
| 📊 **Feature Engineering** | Real-time calculation of CPU, latency, error rates, and composite health scores |
| 🎯 **Policy Engine** | YAML-based DSL for defining operational rules with priorities and cooldowns |
| 🔮 **Simulation** | Monte Carlo projections 5-15 minutes into the future for proactive decisions |
| ⚡ **Action Execution** | Webhook-based action delivery with dry-run and scheduling support |
| 🛡️ **Feedback Loop** | Automatic impact analysis, drift detection, and rollback recommendations |
| 📜 **Full Audit Trail** | Every decision traceable with context, features, and policy version |

### Ideal Use Cases

- **Auto-scaling** infrastructure based on predictive load
- **Circuit breaker** automation for failing services
- **Rate limiting** based on real-time capacity
- **Cost optimization** through simulation-driven decisions
- **Incident response** automation with human-in-the-loop options

---

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- Make

### 1. Clone and Setup

```bash
git clone https://github.com/19hael/Aegis-Decision-Engine-ADE-.git
cd Aegis-Decision-Engine-ADE-
```

### 2. Start Infrastructure

```bash
make up
```

This starts PostgreSQL, Redis, Redpanda (Kafka), Prometheus, and Grafana.

### 3. Run Migrations

```bash
make migrate-up
```

### 4. Start the Server

```bash
make run
```

The server will start on `http://localhost:8080`.

### 5. Run the Demo

```bash
./scripts/demo.sh
```

This demonstrates the complete pipeline: ingest → features → simulation → decision → action → feedback.

---

## Architecture

ADE follows a modular, event-driven architecture with clear boundaries between services.

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  Ingest  │───▶│  State   │───▶│ Decision │───▶│  Action  │
│ Service  │    │ Service  │    │ Service  │    │ Service  │
└──────────┘    └──────────┘    └────┬─────┘    └──────────┘
       │              │              │
       │              │       ┌─────▼─────┐    ┌──────────┐
       │              │       │Simulation │    │ Feedback │
       ▼              ▼       │ Service   │    │ Service  │
  ┌──────────────────────────────────────────────────────────┐
  │              PostgreSQL (Events, Decisions, Traces)       │
  └──────────────────────────────────────────────────────────┘
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed design documentation.

---

## API Reference

### Ingest Events

```bash
# Single event
curl -X POST http://localhost:8080/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "evt-001",
    "idempotency_key": "idemp-001",
    "service_id": "api-gateway",
    "event_type": "metrics",
    "payload": {
      "cpu": 75.5,
      "latency_ms": 450,
      "error_rate": 0.05,
      "requests_per_second": 950,
      "queue_depth": 15
    },
    "timestamp": "2026-02-01T10:00:00Z"
  }'

# Batch events
curl -X POST http://localhost:8080/ingest/batch \
  -H "Content-Type: application/json" \
  -d '[{...}, {...}]'
```

### Calculate Features

```bash
curl -X POST "http://localhost:8080/services/api-gateway/features/calculate?window=5m"
```

Response:
```json
{
  "service_id": "api-gateway",
  "cpu_current": 75.5,
  "cpu_avg_5m": 70.2,
  "latency_p95": 450.0,
  "error_rate": 0.05,
  "health_score": 0.75,
  "load_score": 0.65,
  "throttling_risk": 0.3
}
```

### Run Simulation

```bash
curl -X POST http://localhost:8080/simulations/run \
  -H "Content-Type: application/json" \
  -d '{
    "service_id": "api-gateway",
    "scenario": "high_load",
    "horizon_minutes": 10,
    "iterations": 1000,
    "current_state": { "cpu_current": 75.5, ... }
  }'
```

Response:
```json
{
  "run_id": "sim-123",
  "risk_score": 0.65,
  "recommendation": "scale_up_prepare",
  "aggregates": {
    "probability_overload": 0.45,
    "probability_high_latency": 0.60
  }
}
```

### Make Decision

```bash
curl -X POST http://localhost:8080/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "service_id": "api-gateway",
    "features": {
      "cpu_current": 75.5,
      "latency_p95": 450.0,
      "error_rate": 0.05
    },
    "dry_run": true
  }'
```

### Execute Action

```bash
curl -X POST http://localhost:8080/actions/execute \
  -H "Content-Type: application/json" \
  -d '{
    "action_id": "act-001",
    "action_type": "scale_up",
    "target_service": "api-gateway",
    "payload": {"instances": 2},
    "dry_run": false
  }'
```

### Record Feedback

```bash
curl -X POST http://localhost:8080/feedback \
  -H "Content-Type: application/json" \
  -d '{
    "action_id": "act-001",
    "metrics_before": {"cpu": 75.5, "latency": 450},
    "metrics_after": {"cpu": 55.0, "latency": 320}
  }'
```

---

## Demo

The interactive demo showcases the complete decision pipeline:

```bash
./scripts/demo.sh
```

**What it does:**
1. Ingests 5 simulated metrics events
2. Calculates service features
3. Runs Monte Carlo simulation (10 min projection)
4. Evaluates policy and makes decision
5. Executes action (dry-run mode)
6. Records feedback and analyzes impact

Expected output:
```
╔══════════════════════════════════════════════════════════════╗
║     AEGIS DECISION ENGINE (ADE) - Interactive Demo           ║
╚══════════════════════════════════════════════════════════════╝
✅ Server is healthy
═══════════════════════════════════════════════════════════════
  STEP 1: Ingesting Metrics Events
═══════════════════════════════════════════════════════════════
  Event 1: CPU=73%, Latency=421ms → accepted
  Event 2: CPU=85%, Latency=380ms → accepted
  ...

Risk Score: 0.65
Recommendation: scale_up_prepare

Decision: allow
Confidence: 0.85

✅ Full pipeline executed successfully!
```

---

## Policy DSL

Define operational rules in YAML:

```yaml
id: autoscale_policy
version: "1.0"
name: Auto-Scaling Policy
type: autoscale

rules:
  - id: emergency_scale_up
    name: Emergency Scale Up
    priority: 100
    when:
      any:
        - fact: CPUCurrent
          op: ">="
          value: 90
    action:
      type: scale_up
      cost: 50.0
      risk: 0.2
    cooldown: 5m

  - id: high_load_scale_up
    name: High Load Scale Up
    priority: 80
    when:
      all:
        - fact: CPUCurrent
          op: ">="
          value: 70
        - fact: LatencyP95
          op: ">="
          value: 500
    action:
      type: scale_up
      cost: 30.0
```

See `policies/autoscale_v1.yaml` for a complete example.

---

## Development

### Project Structure

```
aegis-decision-engine/
├── cmd/ade-server/          # Main application entry
├── internal/
│   ├── ingest/              # Event ingestion
│   ├── state/               # Feature calculation
│   ├── decision/            # Policy evaluation
│   ├── simulation/          # Monte Carlo simulation
│   ├── action/              # Action execution
│   ├── feedback/            # Drift detection & rollback
│   ├── policy/              # DSL parser & engine
│   ├── storage/             # PostgreSQL, Redis, Kafka clients
│   └── models/              # Domain models
├── policies/                # Policy YAML files
├── migrations/              # Database migrations
├── deployments/             # Docker Compose, Prometheus, Grafana
└── scripts/                 # Demo and utility scripts
```

### Commands

```bash
make build       # Build binary
make run         # Run server
make test        # Run tests
make lint        # Run linter
make up          # Start infrastructure
make down        # Stop infrastructure
make migrate-up  # Apply migrations
```

### Running Tests

```bash
# Unit tests
go test ./internal/... -v

# Integration tests (requires infrastructure)
make up
go test ./... -tags=integration -v
```

---

## Observability

### Prometheus Metrics

Access Prometheus at `http://localhost:9090`

Key metrics:
- `ade_decisions_total` - Counter of decisions by result
- `ade_decision_duration_seconds` - Histogram of decision latency
- `ade_simulations_total` - Counter of simulations run
- `ade_actions_executed_total` - Counter of actions by status

### Grafana Dashboards

Access Grafana at `http://localhost:3000` (admin/admin)

Pre-configured dashboards:
- ADE Overview - System health and throughput
- Decision Analysis - Decision rates and outcomes
- Simulation Results - Risk scores and recommendations

### Structured Logging

All logs are in JSON format:

```json
{
  "time": "2026-02-01T10:00:00Z",
  "level": "INFO",
  "msg": "decision made",
  "decision_id": "dec-123",
  "service_id": "api-gateway",
  "result": "allow",
  "duration_ms": 15
}
```

---

## Configuration

Configuration via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `ADE_PORT` | 8080 | HTTP server port |
| `DATABASE_URL` | postgres://ade:ade@localhost:5432/ade | PostgreSQL connection |
| `REDIS_URL` | redis://localhost:6379/0 | Redis connection |
| `KAFKA_BROKERS` | localhost:9092 | Kafka/Redpanda brokers |
| `ADE_LOG_LEVEL` | info | Log level (debug, info, warn, error) |

---

## Roadmap

- [x] Core services (Ingest, State, Decision, Simulation, Action, Feedback)
- [x] Policy DSL with YAML
- [x] Monte Carlo simulation
- [x] Drift detection and rollback
- [x] REST API
- [ ] gRPC API
- [ ] OpenTelemetry tracing
- [ ] ML model integration
- [ ] WebSocket real-time streaming
- [ ] Multi-region support
- [ ] Cost optimization engine

---

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details.

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

- Built with ❤️ using Go
- Inspired by event sourcing and CQRS patterns
- Monte Carlo simulation based on statistical methods for operational research

---

<div align="center">

**[⬆ Back to Top](#aegis-decision-engine-ade)**

Made with ⚡ by the ADE Team

</div>
