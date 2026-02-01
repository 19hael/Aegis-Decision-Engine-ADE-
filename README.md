# Aegis Decision Engine (ADE)

Autonomous, auditable, simulation-driven decisioning for real-time systems.

## Quick Start

```bash
# Start infrastructure
make up

# Run migrations
make migrate-up

# Run the server
make run
```

## Database Schema

The system uses PostgreSQL with the following tables:

| Table | Description |
|-------|-------------|
| `events` | All incoming events (event-sourcing light) |
| `feature_snapshots` | Calculated features per service |
| `policies` | Versioned policy definitions in DSL YAML |
| `decision_records` | Immutable decision records |
| `decision_traces` | Decision audit trails |
| `simulation_runs` | Monte Carlo simulation runs |
| `action_records` | Executed or scheduled actions |
| `feedback_records` | Post-action feedback and drift detection |

### Run Migrations

```bash
# Apply migrations
make migrate-up

# Rollback migrations
make migrate-down

# Create new migration
make migrate-create
```

## API Endpoints

- `GET /health` - Health check
- `GET /ready` - Readiness check
- `POST /ingest` - Ingest events

## Development

```bash
make build    # Build binary
make test     # Run tests
make lint     # Run linter
make up       # Start infrastructure (Docker)
make down     # Stop infrastructure
make dev      # Start full dev environment
```
