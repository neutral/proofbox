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
	go test -v -race ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@mkdir -p coverage
	go test -coverprofile=coverage/coverage.out -covermode=atomic ./...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "Coverage report generated at coverage/coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

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
	@echo "  make build        - Build the binary"
	@echo "  make test         - Run tests"
	@echo "  make coverage     - Run tests with coverage report"
	@echo "  make bench        - Run benchmarks"
	@echo "  make lint         - Run linters"
	@echo "  make fmt          - Format code"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make install-tools - Install required development tools"
	@echo "  make all          - Run fmt, test, and lint (default)"