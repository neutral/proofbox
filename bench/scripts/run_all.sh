#!/bin/bash
# run_all.sh - Run all ProofBox benchmarks

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BENCH_DIR="$PROJECT_ROOT/bench"
RESULTS_DIR="$BENCH_DIR/results"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "ProofBox Benchmark Suite"
echo "======================="
echo ""

# Ensure results directory exists
mkdir -p "$RESULTS_DIR"

# Generate timestamp for this run
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
RESULT_FILE="$RESULTS_DIR/benchmark_${TIMESTAMP}.txt"
JSON_FILE="$RESULTS_DIR/benchmark_${TIMESTAMP}.json"

echo "Running benchmarks at $(date)"
echo "Results will be saved to:"
echo "  - $RESULT_FILE"
echo "  - $JSON_FILE"
echo ""

# Change to project root
cd "$PROJECT_ROOT"

# Run benchrunner with both text and JSON output
echo "Starting benchmark run..."
echo ""

# Run benchmarks and save text output
# Use environment variable for benchtime, default to 10s (or 1s in quick mode)
if [ "$BENCH_QUICK" = "1" ]; then
    BENCHTIME="${BENCHTIME:-1s}"
    echo "Quick mode enabled: Using shorter benchmark time"
else
    BENCHTIME="${BENCHTIME:-10s}"
fi

# Use environment variable for benchmark pattern, default to all
BENCH_PATTERN="${BENCH_PATTERN:-.}"

# For quick runs, skip the heaviest system benchmarks
if [ "$BENCH_QUICK" = "1" ] && [ "$BENCH_PATTERN" = "." ]; then
    echo "Quick mode: Skipping heavy system benchmarks (use BENCH_PATTERN to override)"
    BENCH_PATTERN="Benchmark[^R]"  # Skip BenchmarkRealWorld* etc
fi

go run ./bench/cmd/benchrunner/main.go \
    -v \
    -benchtime="$BENCHTIME" \
    -bench="$BENCH_PATTERN" \
    -format=text \
    -output="$RESULT_FILE"

# Run again for JSON output (faster since results are cached)
go run ./bench/cmd/benchrunner/main.go \
    -benchtime=1s \
    -format=json \
    -output="$JSON_FILE"

echo ""
echo -e "${GREEN}✓ Benchmarks completed successfully${NC}"
echo ""

# Display summary
echo "Summary of Results:"
echo "==================="

# Extract key metrics from the results
if command -v jq >/dev/null 2>&1; then
    echo ""
    echo "System Info:"
    jq -r '.system_info | "  Go Version: \(.go_version)\n  Platform: \(.goos)/\(.goarch)\n  CPUs: \(.num_cpu)"' "$JSON_FILE"
    
    echo ""
    echo "Benchmark Count: $(jq '.results | length' "$JSON_FILE")"
    echo "Packages Tested: $(jq '.package_count' "$JSON_FILE")"
    echo "Total Time: $(jq -r '.total_time' "$JSON_FILE")"
    
    echo ""
    echo "Top 5 Fastest Operations:"
    jq -r '.results | sort_by(.ns_per_op) | .[0:5] | .[] | "  \(.name): \(.ns_per_op) ns/op"' "$JSON_FILE"
else
    # Fallback if jq is not available
    tail -n 20 "$RESULT_FILE"
fi

echo ""
echo "Full results available in:"
echo "  - Text: $RESULT_FILE"
echo "  - JSON: $JSON_FILE"
echo ""

# Create a symlink to latest results
ln -sf "benchmark_${TIMESTAMP}.txt" "$RESULTS_DIR/latest.txt"
ln -sf "benchmark_${TIMESTAMP}.json" "$RESULTS_DIR/latest.json"

echo "Latest results symlinked to:"
echo "  - $RESULTS_DIR/latest.txt"
echo "  - $RESULTS_DIR/latest.json"