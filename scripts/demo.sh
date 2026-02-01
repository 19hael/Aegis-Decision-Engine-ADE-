#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

ADE_URL="http://localhost:8080"
SERVICE_ID="api-gateway-prod"

echo -e "${BLUE}"
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║     AEGIS DECISION ENGINE (ADE) - Interactive Demo           ║"
echo "║     Autonomous, Auditable, Simulation-Driven Decisioning     ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Function to make API calls
call_api() {
    local method=$1
    local endpoint=$2
    local data=$3
    
    if [ -n "$data" ]; then
        curl -s -X "$method" "${ADE_URL}${endpoint}" \
            -H "Content-Type: application/json" \
            -d "$data"
    else
        curl -s -X "$method" "${ADE_URL}${endpoint}"
    fi
}

# Function to print section headers
print_section() {
    echo ""
    echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${YELLOW}  $1${NC}"
    echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
}

# Check if server is running
echo -e "${BLUE}Checking if ADE server is running...${NC}"
if ! curl -s "${ADE_URL}/health" > /dev/null 2>&1; then
    echo -e "${RED}❌ ADE server is not running!${NC}"
    echo "Please start it with: make run"
    exit 1
fi
echo -e "${GREEN}✅ Server is healthy${NC}"

# ============================================
# STEP 1: INGEST METRICS EVENTS
# ============================================
print_section "STEP 1: Ingesting Metrics Events"

echo "Simulating incoming metrics from ${SERVICE_ID}..."

# Generate some events with varying CPU/load
for i in {1..5}; do
    CPU=$((50 + RANDOM % 40))  # 50-90% CPU
    LATENCY=$((100 + RANDOM % 400))  # 100-500ms
    ERROR_RATE=$(echo "scale=2; $((RANDOM % 10)) / 100" | bc -l)
    RPS=$((800 + RANDOM % 400))
    QUEUE=$((RANDOM % 20))
    
    EVENT_ID="evt-$(date +%s)-$i"
    
    RESPONSE=$(call_api "POST" "/ingest" "{
        \"event_id\": \"${EVENT_ID}\",
        \"idempotency_key\": \"idemp-${EVENT_ID}\",
        \"service_id\": \"${SERVICE_ID}\",
        \"event_type\": \"metrics\",
        \"payload\": {
            \"cpu\": ${CPU},
            \"latency_ms\": ${LATENCY},
            \"error_rate\": ${ERROR_RATE},
            \"requests_per_second\": ${RPS},
            \"queue_depth\": ${QUEUE}
        },
        \"timestamp\": \"$(date -Iseconds)\"
    }")
    
    echo "  Event $i: CPU=${CPU}%, Latency=${LATENCY}ms → $(echo $RESPONSE | grep -o '"status":"[^"]*"' | cut -d'"' -f4)"
done

echo -e "${GREEN}✅ Ingested 5 metrics events${NC}"

# ============================================
# STEP 2: CALCULATE FEATURES
# ============================================
print_section "STEP 2: Calculating Features"

echo "Computing features for ${SERVICE_ID}..."
FEATURES=$(call_api "POST" "/services/${SERVICE_ID}/features/calculate?window=5m" "")

echo "Features calculated:"
echo "$FEATURES" | python3 -m json.tool 2>/dev/null || echo "$FEATURES"

# ============================================
# STEP 3: RUN SIMULATION
# ============================================
print_section "STEP 3: Running Monte Carlo Simulation"

echo "Projecting 10 minutes ahead with 1000 iterations..."
SIMULATION=$(call_api "POST" "/simulations/run" "{
    \"service_id\": \"${SERVICE_ID}\",
    \"policy_id\": \"autoscale_policy\",
    \"policy_version\": \"1.0\",
    \"snapshot_id\": \"${SERVICE_ID}-current\",
    \"scenario\": \"normal\",
    \"horizon_minutes\": 10,
    \"iterations\": 1000,
    \"current_state\": {
        \"service_id\": \"${SERVICE_ID}\",
        \"cpu_current\": 75.0,
        \"cpu_avg_5m\": 70.0,
        \"latency_p95\": 450.0,
        \"error_rate\": 0.05,
        \"requests_per_second\": 950.0,
        \"queue_depth\": 15,
        \"health_score\": 0.75,
        \"load_score\": 0.65
    }
}")

echo "Simulation results:"
echo "$SIMULATION" | python3 -m json.tool 2>/dev/null | head -50 || echo "$SIMULATION" | head -50

RISK_SCORE=$(echo "$SIMULATION" | grep -o '"risk_score":[0-9.]*' | cut -d':' -f2)
RECOMMENDATION=$(echo "$SIMULATION" | grep -o '"recommendation":"[^"]*"' | cut -d'"' -f4)

echo ""
echo -e "Risk Score: ${YELLOW}${RISK_SCORE}${NC}"
echo -e "Recommendation: ${GREEN}${RECOMMENDATION}${NC}"

# ============================================
# STEP 4: MAKE DECISION
# ============================================
print_section "STEP 4: Making Decision"

echo "Evaluating policy against current features..."
DECISION=$(call_api "POST" "/evaluate" "{
    \"service_id\": \"${SERVICE_ID}\",
    \"policy_id\": \"autoscale_policy\",
    \"features\": {
        \"service_id\": \"${SERVICE_ID}\",
        \"cpu_current\": 75.0,
        \"latency_p95\": 450.0,
        \"error_rate\": 0.05,
        \"requests_per_second\": 950.0,
        \"health_score\": 0.75
    },
    \"dry_run\": true,
    \"idempotency_key\": \"decision-$(date +%s)\"
}")

echo "Decision result:"
echo "$DECISION" | python3 -m json.tool 2>/dev/null || echo "$DECISION"

DECISION_RESULT=$(echo "$DECISION" | grep -o '"result":"[^"]*"' | cut -d'"' -f4)
CONFIDENCE=$(echo "$DECISION" | grep -o '"confidence":[0-9.]*' | cut -d':' -f2)

echo ""
echo -e "Decision: ${GREEN}${DECISION_RESULT}${NC}"
echo -e "Confidence: ${YELLOW}${CONFIDENCE}${NC}"

# ============================================
# STEP 5: EXECUTE ACTION (DRY RUN)
# ============================================
print_section "STEP 5: Executing Action (Dry Run)"

DECISION_ID=$(echo "$DECISION" | grep -o '"decision_id":"[^"]*"' | cut -d'"' -f4)

echo "Executing scale_up action (dry run mode)..."
ACTION=$(call_api "POST" "/actions/execute" "{
    \"action_id\": \"act-${DECISION_ID}\",
    \"decision_id\": \"${DECISION_ID}\",
    \"action_type\": \"scale_up\",
    \"target_service\": \"${SERVICE_ID}\",
    \"payload\": {
        \"instances\": 2,
        \"urgency\": \"normal\"
    },
    \"dry_run\": true
}")

echo "Action result:"
echo "$ACTION" | python3 -m json.tool 2>/dev/null || echo "$ACTION"

# ============================================
# STEP 6: RECORD FEEDBACK
# ============================================
print_section "STEP 6: Recording Feedback"

echo "Recording post-action feedback..."
FEEDBACK=$(call_api "POST" "/feedback" "{
    \"action_id\": \"act-${DECISION_ID}\",
    \"decision_id\": \"${DECISION_ID}\",
    \"service_id\": \"${SERVICE_ID}\",
    \"feedback_type\": \"immediate\",
    \"metrics_before\": {
        \"cpu\": 75.0,
        \"latency\": 450.0,
        \"error_rate\": 0.05,
        \"throughput\": 950.0
    },
    \"metrics_after\": {
        \"cpu\": 55.0,
        \"latency\": 320.0,
        \"error_rate\": 0.03,
        \"throughput\": 980.0
    },
    \"observation_window_minutes\": 5
}")

echo "Feedback analysis:"
echo "$FEEDBACK" | python3 -m json.tool 2>/dev/null || echo "$FEEDBACK"

IMPACT=$(echo "$FEEDBACK" | grep -o '"impact_score":[0-9.-]*' | cut -d':' -f2)
DRIFT=$(echo "$FEEDBACK" | grep -o '"drift_detected":[a-z]*' | cut -d':' -f2)
ROLLBACK=$(echo "$FEEDBACK" | grep -o '"rollback_recommended":[a-z]*' | cut -d':' -f2)

echo ""
echo -e "Impact Score: ${GREEN}${IMPACT}${NC} (positive = improvement)"
echo -e "Drift Detected: ${YELLOW}${DRIFT}${NC}"
echo -e "Rollback Recommended: ${ROLLBACK}${NC}"

# ============================================
# SUMMARY
# ============================================
print_section "Demo Complete!"

echo -e "${GREEN}✅ Full pipeline executed successfully!${NC}"
echo ""
echo "Summary:"
echo "  • Ingested: 5 metrics events"
echo "  • Calculated: Service features (CPU, latency, health scores)"
echo "  • Simulated: 10-min Monte Carlo projection"
echo "  • Decided: ${DECISION_RESULT} (confidence: ${CONFIDENCE})"
echo "  • Executed: scale_up action (dry run)"
echo "  • Feedback: Impact score ${IMPACT}, Drift: ${DRIFT}"
echo ""
echo -e "${BLUE}ADE is ready for production workloads!${NC}"
echo ""
echo "Next steps:"
echo "  • Check Prometheus metrics: http://localhost:9090"
echo "  • View Grafana dashboards: http://localhost:3000 (admin/admin)"
echo "  • Run with real data: ./bin/ade-server"
echo "  • Run tests: make test"
