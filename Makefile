.PHONY: all build test lint fmt clean coverage bench help

# Default target
all: fmt test lint

# Build the binary
build:
	@echo "Building proofbox..."
	@mkdir -p bin
	go build -v -o bin/proofbox cmd/proofbox/main.go

# Run tests
test:
	@echo "Running tests..."
	go test -v -race $$(go list ./... | grep -v '/examples$$')

# Run all tests including long-running performance tests
test-long:
	@echo "Running all tests including performance validation (this may take several minutes)..."
	go test -v -timeout=10m $$(go list ./... | grep -v '/examples$$')

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@mkdir -p coverage
	go test -coverprofile=coverage/coverage.out -covermode=atomic $$(go list ./... | grep -v '/examples$$')
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "Coverage report generated at coverage/coverage.html"

# Run all benchmarks (both system and package level)
bench-all:
	@echo "Running all benchmarks..."
	@bash ./bench/scripts/run_all.sh

# Run quick benchmarks for development
bench-quick:
	@echo "Running quick benchmarks for development..."
	@bash ./bench/scripts/quick_bench.sh

# Run only system benchmarks
bench-system:
	@echo "Running system benchmarks..."
	@go test -bench=. -benchmem ./bench/... | tee bench/results/system_$(shell date +%Y%m%d_%H%M%S).txt

# Run only package benchmarks  
bench-pkg:
	@echo "Running package benchmarks..."
	@BENCHTIME="${BENCHTIME:-1s}" go test -bench=. -benchmem $$(go list ./... | grep -v -E '/(bench|examples)$$') | tee bench/results/pkg_$(shell date +%Y%m%d_%H%M%S).txt

# Run benchmarks (short version for backwards compatibility)
bench: bench-system

# Run large-scale benchmarks with full datasets
bench-large:
	@echo "Running large-scale benchmarks (this may take several minutes)..."
	@BENCH_LARGE=1 BENCHTIME=30s bash ./bench/scripts/run_all.sh

# Validate performance targets
bench-validate:
	@echo "Validating performance targets..."
	@bash ./bench/scripts/validate_targets.sh

# Generate performance report
bench-report:
	@echo "Generating performance report..."
	@bash ./bench/scripts/generate_report.sh

# Run linter
lint:
	@echo "Running linters..."
	golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

# Clean build artifacts
clean:
	@echo "Cleaning..."
	go clean -testcache
	rm -rf bin/ coverage/

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Show help
help:
	@echo "Available targets:"
	@echo "  make build         - Build the binary"
	@echo "  make test          - Run tests (excludes examples)"
	@echo "  make test-long     - Run all tests including performance validation"
	@echo "  make coverage      - Run tests with coverage report"
	@echo "  make bench         - Run system benchmarks (short)"
	@echo "  make bench-quick   - Run quick benchmarks for development"
	@echo "  make bench-all     - Run all benchmarks (system + package)"
	@echo "  make bench-large   - Run large-scale benchmarks with full datasets"
	@echo "  make bench-system  - Run only system-level benchmarks"
	@echo "  make bench-pkg     - Run only package-level benchmarks"
	@echo "  make bench-validate - Validate performance targets"
	@echo "  make bench-report  - Generate performance report"
	@echo "  make lint          - Run linters"
	@echo "  make fmt           - Format code"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make install-tools - Install required development tools"
	@echo "  make all           - Run fmt, test, and lint (default)"