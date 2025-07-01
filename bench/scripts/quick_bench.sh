#!/bin/bash
# quick_bench.sh - Quick benchmark runner for development testing

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BENCH_DIR="$PROJECT_ROOT/bench"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "ProofBox Quick Benchmarks"
echo "========================"
echo ""

# Change to project root
cd "$PROJECT_ROOT"

# Set environment variables for quick mode
export BENCH_QUICK=1
export BENCHTIME=1s

echo -e "${YELLOW}Quick mode enabled:${NC}"
echo "  - Tree sizes limited to 1K keys for setup"
echo "  - Benchmark time: 1s per test"
echo "  - Large scenarios skipped"
echo ""

# Run quick benchmarks
echo "Running quick benchmarks..."
go test -bench=. -benchtime=1s -short ./bench/...

echo ""
echo -e "${GREEN}✓ Quick benchmarks completed${NC}"
echo ""
echo "For full benchmarks with larger datasets, run:"
echo "  make bench-all"
echo "  BENCH_LARGE=1 make bench-system"