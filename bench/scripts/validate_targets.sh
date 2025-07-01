#!/bin/bash
# validate_targets.sh - Validate that performance targets are met

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BENCH_DIR="$PROJECT_ROOT/bench"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "ProofBox Performance Target Validation"
echo "====================================="
echo ""

# Change to project root
cd "$PROJECT_ROOT"

# Function to check if a test passed
check_test() {
    local test_name=$1
    local test_output=$2
    
    if echo "$test_output" | grep -q "PASS.*$test_name"; then
        echo -e "${GREEN}✓ $test_name: PASSED${NC}"
        return 0
    else
        echo -e "${RED}✗ $test_name: FAILED${NC}"
        # Extract failure reason if available
        echo "$test_output" | grep -A5 "$test_name" | grep -E "(FAIL|Error:|not ok)" | sed 's/^/    /'
        return 1
    fi
}

# Function to extract metric from test output
extract_metric() {
    local test_output=$1
    local pattern=$2
    echo "$test_output" | grep -E "$pattern" | tail -1
}

echo "Running performance validation tests..."
echo ""

# Run the requirements tests
TEST_OUTPUT=$(go test -v -timeout=30m ./bench -run "^Test.*Requirement" 2>&1) || true

# Track overall pass/fail
FAILED=0

# Check each requirement test
echo "Performance Requirements:"
echo "------------------------"

# 1. Insert Throughput
if check_test "TestInsertThroughputRequirement" "$TEST_OUTPUT"; then
    metric=$(extract_metric "$TEST_OUTPUT" "Insert throughput:.*ops/sec")
    [ -n "$metric" ] && echo "    $metric"
else
    FAILED=$((FAILED + 1))
fi

# 2. Lookup Latency
if check_test "TestLookupLatencyRequirement" "$TEST_OUTPUT"; then
    metric=$(extract_metric "$TEST_OUTPUT" "Lookup latency p95:.*ms")
    [ -n "$metric" ] && echo "    $metric"
else
    FAILED=$((FAILED + 1))
fi

# 3. Proof Generation Latency
if check_test "TestProofGenerationLatency" "$TEST_OUTPUT"; then
    metric=$(extract_metric "$TEST_OUTPUT" "Proof generation latency p95:.*ms")
    [ -n "$metric" ] && echo "    $metric"
else
    FAILED=$((FAILED + 1))
fi

# 4. Proof Verification Latency
if check_test "TestProofVerificationLatency" "$TEST_OUTPUT"; then
    metric=$(extract_metric "$TEST_OUTPUT" "Proof verification latency p95:.*μs")
    [ -n "$metric" ] && echo "    $metric"
else
    FAILED=$((FAILED + 1))
fi

# 5. Batch Commit Linear Scaling
if check_test "TestBatchCommitLinearScaling" "$TEST_OUTPUT"; then
    echo "    Batch commit scales linearly ✓"
else
    FAILED=$((FAILED + 1))
fi

echo ""
echo "System Information:"
echo "------------------"
extract_metric "$TEST_OUTPUT" "System Information:" | head -n 6 | tail -n 5 | sed 's/^/  /'

echo ""
echo "Summary:"
echo "--------"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All performance targets met!${NC}"
    echo ""
    echo "ProofBox meets the following requirements:"
    echo "  • Insert throughput ≥ 50,000 ops/sec"
    echo "  • Lookup latency p95 ≤ 1ms at scale"
    echo "  • Proof generation p95 ≤ 5ms"
    echo "  • Proof verification p95 ≤ 300μs"
    echo "  • Batch commit scales linearly"
    exit 0
else
    echo -e "${RED}✗ $FAILED performance targets not met${NC}"
    echo ""
    echo "Performance requirements:"
    echo "  • Insert throughput must be ≥ 50,000 ops/sec"
    echo "  • Lookup latency p95 must be ≤ 1ms at scale"
    echo "  • Proof generation p95 must be ≤ 5ms"
    echo "  • Proof verification p95 must be ≤ 300μs"
    echo "  • Batch commit must scale linearly"
    echo ""
    echo "Note: Performance may vary based on hardware. Requirements assume 8-core 2024 laptop."
    exit 1
fi