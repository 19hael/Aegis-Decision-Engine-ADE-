#!/bin/bash
set -e

echo "=== ADE Database Migration Test ==="
echo ""

# Check if docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running"
    exit 1
fi

echo "1. Starting PostgreSQL..."
docker-compose -f deployments/docker-compose.yaml up -d postgres

echo "2. Waiting for PostgreSQL to be ready..."
sleep 3
until docker exec ade-postgres pg_isready -U ade > /dev/null 2>&1; do
    echo "   Waiting..."
    sleep 1
done
echo "   ✅ PostgreSQL is ready"

echo ""
echo "3. Running migrations UP..."
migrate -path migrations -database "postgres://ade:ade@localhost:5432/ade?sslmode=disable" up

echo ""
echo "4. Verifying tables..."
docker exec ade-postgres psql -U ade -d ade -c "\dt" | grep -E "(events|feature_snapshots|policies|decision_records|decision_traces|simulation_runs|action_records|feedback_records)" | while read line; do
    echo "   ✅ $line"
done

echo ""
echo "5. Migration test completed successfully!"
echo ""
echo "To rollback: make migrate-down"
echo "To create new migration: make migrate-create"
