---
id: step.01.project‑scaffold
depends_on: []
tags: [setup, step]
---

## Objective

Create Go module `github.com/neutral/proofbox` with project structure, development environment, and initial CI/CD setup.

## Implements

Establishing the Go module and a "green" `go test ./...` run satisfies the **Scope & Assumptions** clause that "this spec can guide an automated code‑generation phase for implementations in Rust **or other languages**." Before any JMT‑specific logic exists, a deterministic build pipeline is needed to guarantee repeatability and CI hooks (§Architecture Overview, final paragraph).

## Technical Details

### Module Initialization
```bash
go mod init github.com/neutral/proofbox
go mod tidy
```

### Directory Structure
```
.
├── .github/
│   └── workflows/
│       └── ci.yml          # GitHub Actions CI workflow
├── blueprint/             # Project specifications (existing)
├── cmd/
│   └── proofbox/          # CLI application
│       └── main.go
├── genkit/               # Generation toolkit (existing)
├── internal/              # Internal packages
│   └── .gitkeep
├── pkg/                   # Public packages
│   ├── crypto/           # Cryptographic functions
│   ├── proof/            # Proof generation/verification
│   ├── storage/          # Storage abstraction
│   ├── tree/             # Core tree implementation
│   └── types/            # Common types
├── scripts/              # Build and development scripts
│   └── .gitkeep
├── docs/                 # Additional documentation
│   ├── api/             # API documentation
│   ├── design/          # Design documents
│   └── examples/        # Usage examples
├── .gitignore
├── .golangci.yml         # Linting configuration
├── DEVELOPMENT.md        # Development guide
├── Makefile              # Build automation
├── README.md             # Project documentation
└── go.mod
```

### Development Tools Setup

1. **Linting Configuration** (`.golangci.yml`):
```yaml
linters:
  enable:
    - gofmt
    - revive        # Replaces deprecated golint
    - govet
    - ineffassign
    - misspell
    - unconvert
    - prealloc
    - nakedret
    - gocritic
    - staticcheck   # Advanced static analysis
    - gosimple      # Simplification suggestions
    - unused        # Finds unused code
    - errcheck      # Checks for unchecked errors
    - gosec         # Security issues
    - goimports     # Import formatting

run:
  timeout: 10m
  skip-dirs:
    - vendor
    - genkit
    - blueprint
```

2. **Makefile**:
```makefile
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
```

3. **Git Hooks** (`.githooks/pre-commit`):
```bash
#!/bin/sh
make lint
make test
```

## Implementation Steps

1. Initialize Go module with appropriate module path
2. Create directory structure with placeholder files
3. Set up development tooling (linting, formatting)
4. Configure CI/CD pipeline for automated testing
5. Add basic documentation templates

## Testing Requirements

- Verify module initialization succeeds
- Ensure directory structure is created correctly
- Confirm CI pipeline runs successfully on empty project
- Validate linting configuration works

## Done When ✓

- [x] Go module initialized with correct path
- [x] Directory structure created with all folders
- [x] `.golangci.yml` configured with modern linters (revive, staticcheck, etc.)
- [x] `Makefile` with build, test, coverage, lint, fmt, and install-tools targets
- [x] GitHub Actions CI workflow configured
- [x] `go test ./...` passes with no packages
- [x] `make build` creates binary in bin/
- [x] `make coverage` generates coverage report
- [x] `make lint` runs successfully (if tools installed)
- [x] DEVELOPMENT.md with contribution guidelines
- [x] docs/ directory structure for documentation
- [x] README.md with basic project information
