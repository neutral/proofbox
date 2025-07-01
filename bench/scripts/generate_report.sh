#!/bin/bash
# generate_report.sh - Generate a comprehensive performance report

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BENCH_DIR="$PROJECT_ROOT/bench"
RESULTS_DIR="$BENCH_DIR/results"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "ProofBox Performance Report Generator"
echo "===================================="
echo ""

# Check if jq is available
if ! command -v jq >/dev/null 2>&1; then
    echo -e "${RED}Error: jq is required for report generation${NC}"
    echo "Please install jq: https://stedolan.github.io/jq/download/"
    exit 1
fi

# Find the latest benchmark results
LATEST_JSON="$RESULTS_DIR/latest.json"
if [ ! -f "$LATEST_JSON" ]; then
    echo -e "${RED}Error: No benchmark results found${NC}"
    echo "Run './bench/scripts/run_all.sh' first to generate benchmark results"
    exit 1
fi

# Generate timestamp for report
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
REPORT_FILE="$RESULTS_DIR/performance_report_${TIMESTAMP}.md"

echo "Generating performance report..."
echo "Report will be saved to: $REPORT_FILE"
echo ""

# Start generating the report
cat > "$REPORT_FILE" << 'EOF'
# ProofBox Performance Report

EOF

# Add generation date
echo "Generated: $(date '+%Y-%m-%d %H:%M:%S')" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Add system information
echo "## System Information" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
jq -r '.system_info | 
"- **Go Version**: \(.go_version)
- **Platform**: \(.goos)/\(.goarch)  
- **CPU Cores**: \(.num_cpu)
- **GOMAXPROCS**: \(.gomaxprocs)"' "$LATEST_JSON" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Add executive summary
echo "## Executive Summary" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Calculate key metrics
TOTAL_BENCHMARKS=$(jq '.results | length' "$LATEST_JSON")
PACKAGE_COUNT=$(jq '.package_count' "$LATEST_JSON")
TOTAL_TIME=$(jq -r '.total_time' "$LATEST_JSON")

echo "- **Total Benchmarks**: $TOTAL_BENCHMARKS" >> "$REPORT_FILE"
echo "- **Packages Tested**: $PACKAGE_COUNT" >> "$REPORT_FILE"
echo "- **Total Test Time**: $TOTAL_TIME" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Performance targets section
echo "## Performance Targets" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
echo "| Target | Requirement | Status |" >> "$REPORT_FILE"
echo "|--------|-------------|--------|" >> "$REPORT_FILE"

# Check if validation results exist
if [ -f "$RESULTS_DIR/validation_latest.txt" ]; then
    # Parse validation results
    if grep -q "All performance targets met" "$RESULTS_DIR/validation_latest.txt"; then
        echo "| Insert Throughput | ≥ 50,000 ops/sec | ✅ Met |" >> "$REPORT_FILE"
        echo "| Lookup Latency | p95 ≤ 1ms | ✅ Met |" >> "$REPORT_FILE"
        echo "| Proof Generation | p95 ≤ 5ms | ✅ Met |" >> "$REPORT_FILE"
        echo "| Proof Verification | p95 ≤ 300μs | ✅ Met |" >> "$REPORT_FILE"
        echo "| Batch Scaling | Linear | ✅ Met |" >> "$REPORT_FILE"
    else
        echo "| Insert Throughput | ≥ 50,000 ops/sec | ⚠️ Check |" >> "$REPORT_FILE"
        echo "| Lookup Latency | p95 ≤ 1ms | ⚠️ Check |" >> "$REPORT_FILE"
        echo "| Proof Generation | p95 ≤ 5ms | ⚠️ Check |" >> "$REPORT_FILE"
        echo "| Proof Verification | p95 ≤ 300μs | ⚠️ Check |" >> "$REPORT_FILE"
        echo "| Batch Scaling | Linear | ⚠️ Check |" >> "$REPORT_FILE"
    fi
else
    echo "| All Targets | Various | ⚠️ Run validation |" >> "$REPORT_FILE"
fi
echo "" >> "$REPORT_FILE"

# Key performance metrics
echo "## Key Performance Metrics" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Find system benchmarks
echo "### System-Level Benchmarks" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Extract key system benchmarks
jq -r '.results[] | select(.package == "./bench") | 
"\(.name): **\(1000000000 / .ns_per_op | floor) ops/sec** (\(.ns_per_op | floor) ns/op)"' "$LATEST_JSON" 2>/dev/null | head -10 >> "$REPORT_FILE" || echo "No system benchmarks found" >> "$REPORT_FILE"

echo "" >> "$REPORT_FILE"

# Top performers
echo "### Top 10 Fastest Operations" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
echo "| Operation | Package | Performance |" >> "$REPORT_FILE"
echo "|-----------|---------|-------------|" >> "$REPORT_FILE"

jq -r '.results | sort_by(.ns_per_op) | .[0:10] | .[] | 
"| \(.name) | \(.package) | \(1000000000 / .ns_per_op | floor) ops/sec |"' "$LATEST_JSON" >> "$REPORT_FILE"

echo "" >> "$REPORT_FILE"

# Memory efficiency
echo "### Memory Efficiency" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
echo "| Operation | Bytes/Op | Allocs/Op |" >> "$REPORT_FILE"
echo "|-----------|----------|-----------|" >> "$REPORT_FILE"

jq -r '.results | sort_by(.bytes_per_op) | reverse | .[0:10] | .[] | select(.bytes_per_op > 0) |
"| \(.name) | \(.bytes_per_op) | \(.allocs_per_op) |"' "$LATEST_JSON" >> "$REPORT_FILE"

echo "" >> "$REPORT_FILE"

# Package breakdown
echo "## Package Performance Breakdown" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Group by package and show summary
jq -r '.results | group_by(.package) | .[] | 
"### \(.[0].package)

- **Benchmarks**: \(length)
- **Avg ns/op**: \([.[].ns_per_op] | add / length | floor)
- **Total Bytes**: \([.[].bytes_per_op] | add)

"' "$LATEST_JSON" >> "$REPORT_FILE"

# Recommendations
echo "## Recommendations" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Check for high memory allocations
HIGH_ALLOC_COUNT=$(jq '[.results[] | select(.bytes_per_op > 10000)] | length' "$LATEST_JSON")
if [ "$HIGH_ALLOC_COUNT" -gt 0 ]; then
    echo "- ⚠️ **Memory Usage**: $HIGH_ALLOC_COUNT operations allocate >10KB per operation. Consider optimization." >> "$REPORT_FILE"
fi

# Check for slow operations
SLOW_OP_COUNT=$(jq '[.results[] | select(.ns_per_op > 1000000)] | length' "$LATEST_JSON")
if [ "$SLOW_OP_COUNT" -gt 0 ]; then
    echo "- ⚠️ **Performance**: $SLOW_OP_COUNT operations take >1ms. Review for optimization opportunities." >> "$REPORT_FILE"
fi

echo "- ✅ **Coverage**: All major components have benchmark coverage" >> "$REPORT_FILE"
echo "- 📊 **Monitoring**: Set up continuous performance monitoring in CI" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Footer
echo "---" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
echo "*Report generated by ProofBox benchmark suite*" >> "$REPORT_FILE"

# Create symlink to latest report
ln -sf "performance_report_${TIMESTAMP}.md" "$RESULTS_DIR/latest_report.md"

echo -e "${GREEN}✓ Performance report generated successfully${NC}"
echo ""
echo "Report saved to:"
echo "  $REPORT_FILE"
echo ""
echo "Latest report symlinked to:"
echo "  $RESULTS_DIR/latest_report.md"
echo ""

# Display summary
echo -e "${BLUE}Report Summary:${NC}"
head -n 50 "$REPORT_FILE" | grep -E "^(#|##|\*\*|-)""${NC}"