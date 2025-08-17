.PHONY: all build test lint fmt clean coverage bench help fuzz fuzz-one fuzz-rapid fuzz-quick

# Default target
all: fmt test lint

# Build the binary
build:
	@echo "Building pb..."
	@mkdir -p bin
	go build -v -o bin/pb cmd/pb/main.go

# Run tests - fast mode for development and CI commits (<10 minutes)
test:
	@echo "Running fast tests..."
	go test -v -short -timeout=5m $$(go list ./... | grep -v '/examples')

# Run tests with race detector
test-race:
	@echo "Running tests with race detector (short mode)..."
	go test -v -race -short -timeout=10m $$(go list ./... | grep -v '/examples')

# Run minimal tests for CI commits (<5 minutes)
test-commit:
	@echo "Running commit tests..."
	FUZZ_LEVEL=quick go test -v -short -timeout=5m ./pkg/... ./cmd/...

# Run all tests including long-running performance tests
test-long:
	@echo "Running comprehensive tests (this may take 20-30 minutes)..."
	go test -v -race -timeout=30m $$(go list ./... | grep -v '/examples')

# Run all tests without race detector (faster)
test-all:
	@echo "Running all tests without race detector..."
	go test -v -timeout=15m $$(go list ./... | grep -v '/examples')

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@mkdir -p coverage
	go test -coverprofile=coverage/coverage.out -covermode=atomic $$(go list ./... | grep -v '/examples')
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
	@BENCHTIME="${BENCHTIME:-1s}" go test -bench=. -benchmem $$(go list ./... | grep -v -E '/(bench|examples)') | tee bench/results/pkg_$(shell date +%Y%m%d_%H%M%S).txt

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
	@command -v goimports >/dev/null 2>&1 && goimports -w . || echo "goimports not installed, skipping import formatting"

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

# Run property-based tests with rapid (standard: 100 iterations)
fuzz-rapid:
	@echo "Running property-based tests with rapid (standard: 100 iterations)..."
	@mkdir -p $(LOCAL_GOCACHE)
	FUZZ_LEVEL=standard GOCACHE=$(PWD)/$(LOCAL_GOCACHE) go test -v ./pkg/fuzz/properties/... ./pkg/fuzz/performance/... -rapid.checks=100

# Run quick property tests for development (20 iterations)
fuzz-quick:
	@echo "Running quick property tests (20 iterations)..."
	@mkdir -p $(LOCAL_GOCACHE)
	FUZZ_LEVEL=quick GOCACHE=$(PWD)/$(LOCAL_GOCACHE) go test -v -short ./pkg/fuzz/properties/... ./pkg/fuzz/performance/... -rapid.checks=20

# Run comprehensive property tests (1000 iterations)
fuzz-comprehensive:
	@echo "Running comprehensive property tests (1000 iterations)..."
	@mkdir -p $(LOCAL_GOCACHE)
	FUZZ_LEVEL=comprehensive GOCACHE=$(PWD)/$(LOCAL_GOCACHE) go test -v -timeout=1h ./pkg/fuzz/properties/... ./pkg/fuzz/performance/... -rapid.checks=1000

# Run stress property tests (10000 iterations)
fuzz-stress:
	@echo "Running stress property tests (10000 iterations)..."
	@mkdir -p $(LOCAL_GOCACHE)
	FUZZ_LEVEL=stress GOCACHE=$(PWD)/$(LOCAL_GOCACHE) go test -v -timeout=2h ./pkg/fuzz/properties/... ./pkg/fuzz/performance/... -rapid.checks=10000

# Run native Go fuzzing
FUZZTIME ?= 2m
LOCAL_GOCACHE ?= .gocache

fuzz:
	@echo "Running native Go fuzzing..."
ifdef FUZZ
	@echo "Fuzz target: $(FUZZ) for $(FUZZTIME)"
	@mkdir -p $(LOCAL_GOCACHE)
	GOCACHE=$(PWD)/$(LOCAL_GOCACHE) go test -run=^$$ -fuzz=$(FUZZ) -fuzztime=$(FUZZTIME) ./pkg/fuzz
else
	@echo "No FUZZ specified; running both targets for 1m each"
	$(MAKE) --no-print-directory fuzz-one FUZZ=FuzzTreeOperations FUZZTIME=1m LOCAL_GOCACHE=$(LOCAL_GOCACHE)
	$(MAKE) --no-print-directory fuzz-one FUZZ=FuzzProofGeneration FUZZTIME=1m LOCAL_GOCACHE=$(LOCAL_GOCACHE)
endif

fuzz-one:
	@echo "Fuzz target: $(FUZZ) for $(FUZZTIME)"
	@mkdir -p $(LOCAL_GOCACHE)
	GOCACHE=$(PWD)/$(LOCAL_GOCACHE) go test -run=^$$ -fuzz=$(FUZZ) -fuzztime=$(FUZZTIME) ./pkg/fuzz

# Show help
help:
	@echo "Available targets:"
	@echo "  make build         - Build the binary"
	@echo "  make test          - Run fast tests in short mode (<10 min)"
	@echo "  make test-race     - Run tests with race detector in short mode"
	@echo "  make test-commit   - Run minimal commit tests (<5 min)"
	@echo "  make test-long     - Run comprehensive tests with race detector (20-30 min)"
	@echo "  make test-all      - Run all tests without race detector"
	@echo "  make coverage      - Run tests with coverage report"
	@echo "  make bench         - Run system benchmarks (short)"
	@echo "  make bench-quick   - Run quick benchmarks for development"
	@echo "  make bench-all     - Run all benchmarks (system + package)"
	@echo "  make bench-large   - Run large-scale benchmarks with full datasets"
	@echo "  make bench-system  - Run only system-level benchmarks"
	@echo "  make bench-pkg     - Run only package-level benchmarks"
	@echo "  make bench-validate - Validate performance targets"
	@echo "  make bench-report  - Generate performance report"
	@echo "  make fuzz          - Run native Go fuzzing (set FUZZ=FuzzName to target one)"
	@echo "  make fuzz-rapid    - Run property tests with rapid (100 checks)"
	@echo "  make fuzz-quick    - Run quick property tests (20 checks)"
	@echo "  make fuzz-comprehensive - Run comprehensive property tests (1000 checks)"
	@echo "  make fuzz-stress   - Run stress property tests (10000 checks)"
	@echo "  make lint          - Run linters"
	@echo "  make fmt           - Format code"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make install-tools - Install required development tools"
	@echo "  make all           - Run fmt, test, and lint (default)"
