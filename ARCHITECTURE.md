# ADE Architecture

## Overview

Aegis Decision Engine (ADE) is an autonomous, auditable, simulation-driven decision system for real-time operations. It follows an event-sourcing light pattern with full traceability.

```
┌─────────────────────────────────────────────────────────────────┐
│                         ADE Architecture                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐  │
│  │  Ingest  │───▶│  State   │───▶│ Decision │───▶│  Action  │  │
│  │ Service  │    │ Service  │    │ Service  │    │ Service  │  │
│  └──────────┘    └──────────┘    └────┬─────┘    └──────────┘  │
│         │              │              │                          │
│         │              │       ┌─────▼─────┐    ┌──────────┐   │
│         │              │       │Simulation │    │ Feedback │   │
│         │              │       │ Service   │    │ Service  │   │
│         ▼              ▼       └───────────┘    └──────────┘   │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              PostgreSQL (Events, Decisions, Traces)       │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌──────────────────┐        ┌──────────────────────────────┐  │
│  │  Redis (State)   │        │  Redpanda/Kafka (Streaming)  │  │
│  └──────────────────┘        └──────────────────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Core Principles

### 1. Event Sourcing Light
All incoming events are persisted immutable. Decisions and actions are also events that form a complete audit trail.

### 2. Determinism
Decisions are reproducible with:
- `decision_id` 
- `policy_version`
- `feature_snapshot`

### 3. Idempotency
Every event and action has an `idempotency_key` to prevent duplicates.

### 4. Observability
Full tracing with OpenTelemetry, metrics in Prometheus, logs in structured JSON.

## Services

### Ingest Service
**Responsibility:** Receive and validate incoming events

**Inputs:**
- REST: `POST /ingest`, `POST /ingest/batch`
- Kafka: `ade.events` topic

**Outputs:**
- PostgreSQL: `events` table
- Kafka: `ade.events` topic (for downstream processing)

**Key Features:**
- Schema validation (JSON Schema)
- Idempotency checking
- Batch ingestion support

### State Service
**Responsibility:** Calculate and maintain service features

**Endpoints:**
- `GET /services/{id}/state` - Latest features
- `POST /services/{id}/features/calculate` - Compute from events

**Features Calculated:**
- CPU: current, avg (5m, 15m), EMA, trend
- Latency: p50, p95, p99, EMA
- Error rate: current, 5m avg, spike detection
- Throughput: RPS, trend
- Queue: depth, saturation
- Composite: load_score, health_score, throttling_risk

### Decision Service
**Responsibility:** Evaluate policies and make decisions

**Endpoints:**
- `POST /evaluate` - Make a decision
- `GET /decisions/{id}` - Get decision record
- `GET /decisions/{id}/trace` - Get audit trace

**Policy DSL:**
```yaml
rules:
  - id: high_cpu
    priority: 100
    when:
      all:
        - fact: CPUCurrent
          op: ">="
          value: 80
    action:
      type: scale_up
      cost: 10
      risk: 0.1
```

### Simulation Service
**Responsibility:** Monte Carlo projections for what-if analysis

**Endpoint:** `POST /simulations/run`

**Scenarios:**
- `normal` - Standard behavior
- `high_load` - Increased traffic
- `failure` - Error conditions
- `recovery` - System healing

**Outputs:**
- Projected states per minute
- Probability of overload/high latency/errors
- Cost projections
- Risk scores
- Recommendations

### Action Service
**Responsibility:** Execute decisions via webhooks

**Endpoints:**
- `POST /actions/execute` - Execute immediately
- `POST /actions/schedule` - Schedule for later
- `POST /actions/batch` - Execute multiple

**Features:**
- Dry-run mode
- Webhook delivery with retries
- Action queuing
- Execution tracking

### Feedback Service
**Responsibility:** Measure impact and detect drift

**Endpoints:**
- `POST /feedback` - Record post-action metrics
- `POST /rollback` - Execute rollback
- `GET /services/{id}/drift` - Check for drift

**Analysis:**
- Impact score (-1 to +1)
- Drift detection (KS-test heuristic)
- Auto-rollback recommendations
- Severity classification

## Data Model

### Events
```sql
CREATE TABLE events (
    id UUID PRIMARY KEY,
    event_id VARCHAR(255) UNIQUE NOT NULL,
    idempotency_key VARCHAR(255) UNIQUE NOT NULL,
    service_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ
);
```

### Decision Records
```sql
CREATE TABLE decision_records (
    id UUID PRIMARY KEY,
    decision_id VARCHAR(255) UNIQUE NOT NULL,
    service_id VARCHAR(255) NOT NULL,
    policy_id VARCHAR(255) NOT NULL,
    policy_version VARCHAR(50) NOT NULL,
    decision_result VARCHAR(50) NOT NULL,
    actions JSONB NOT NULL,
    confidence_score DECIMAL(5,4),
    executed_at TIMESTAMPTZ NOT NULL
);
```

### Decision Traces
```sql
CREATE TABLE decision_traces (
    id UUID PRIMARY KEY,
    trace_id VARCHAR(255) UNIQUE NOT NULL,
    decision_id VARCHAR(255) NOT NULL,
    trace_data JSONB NOT NULL,
    rules_evaluated JSONB NOT NULL,
    rules_matched JSONB NOT NULL,
    execution_time_ms INTEGER NOT NULL
);
```

## Decision Flow

```
1. Event Ingestion
   └─▶ Validate schema
   └─▶ Check idempotency
   └─▶ Store in PostgreSQL
   └─▶ Publish to Kafka

2. Feature Calculation (async or on-demand)
   └─▶ Read recent events
   └─▶ Calculate aggregates
   └─▶ Store feature snapshot

3. Decision Evaluation
   └─▶ Load policy
   └─▶ Load features
   └─▶ Run simulation (optional)
   └─▶ Evaluate rules (priority order)
   └─▶ Store decision + trace

4. Action Execution
   └─▶ Queue action
   └─▶ Send webhook
   └─▶ Track execution

5. Feedback Loop
   └─▶ Collect post-action metrics
   └─▶ Calculate impact
   └─▶ Detect drift
   └─▶ Recommend rollback if needed
```

## Scalability Considerations

### Horizontal Scaling
- **Ingest Service:** Can scale behind load balancer
- **State Service:** Cache features in Redis
- **Decision Service:** Stateless, can scale horizontally
- **Kafka:** Partition by service_id for parallel processing

### Database Optimization
- Indexes on: service_id, timestamp, decision_id
- Partition events by time (monthly)
- Archive old traces to cold storage

### Caching Strategy
- Features: 5-minute TTL in Redis
- Policies: Load on startup, hot-reload on change
- Decisions: Cache recent in memory

## Security

### Authentication
- API Keys for external endpoints
- mTLS for internal service communication

### Authorization
- RBAC for policy management
- Service-level permissions for actions

### Audit
- All decisions traced with full context
- Immutable event log
- Decision replay capability

## Deployment

### Local Development
```bash
make up     # Start infrastructure
make run    # Run server
make test   # Run tests
```

### Production
- Kubernetes with HPA
- Separate pools for critical vs background workloads
- Prometheus + Grafana for observability
- PagerDuty integration for alerts

## Technology Choices

| Component | Technology | Reason |
|-----------|------------|--------|
| Language | Go 1.22+ | Performance, concurrency, maintainability |
| Database | PostgreSQL 16 | Reliability, JSON support, ecosystem |
| Cache | Redis 7 | Speed, pub/sub for real-time features |
| Streaming | Redpanda | Kafka-compatible, simpler ops |
| Metrics | Prometheus | Industry standard, pull model |
| Tracing | OpenTelemetry | Vendor-neutral, growing ecosystem |
| Config | YAML + Env | Flexibility, 12-factor app compliance |

## Future Enhancements

1. **ML Models:** Pluggable ML for anomaly detection
2. **Multi-Region:** Cross-region decision coordination
3. **GraphQL:** Alternative to REST for complex queries
4. **WebSocket:** Real-time decision streaming
5. **Cost Optimization:** Better cost modeling and prediction
