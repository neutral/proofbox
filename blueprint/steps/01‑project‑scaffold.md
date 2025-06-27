---
id: step.01.project‑scaffold
depends_on: []
tags: [setup, step]
---

## Objective

Create Go module `github.com/acme/jmt` with project structure, development environment, and initial CI/CD setup.

## Implements

Establishing the Go module and a "green" `go test ./...` run satisfies the **Scope & Assumptions** clause that "this spec can guide an automated code‑generation phase for implementations in Rust **or other languages**." Before any JMT‑specific logic exists, a deterministic build pipeline is needed to guarantee repeatability and CI hooks (§Architecture Overview, final paragraph).

## Technical Details

### Module Initialization
```bash
go mod init github.com/acme/jmt
go mod tidy
```

### Directory Structure
```
jmt/
├── .github/
│   └── workflows/
│       └── ci.yml          # GitHub Actions CI workflow
├── cmd/
│   └── jmt/               # CLI application
│       └── .gitkeep
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
├── .gitignore
├── .golangci.yml         # Linting configuration
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
    - golint
    - govet
    - ineffassign
    - misspell
    - unconvert
    - prealloc
    - nakedret
    - gocritic
```

2. **Makefile**:
```makefile
.PHONY: all test lint fmt clean

all: test lint

test:
	go test -v -race ./...

lint:
	golangci-lint run

fmt:
	go fmt ./...

clean:
	go clean -testcache
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

- [ ] Go module initialized with correct path
- [ ] Directory structure created with all folders
- [ ] `.golangci.yml` configured with recommended linters
- [ ] `Makefile` with test, lint, and fmt targets
- [ ] GitHub Actions CI workflow configured
- [ ] `go test ./...` passes with no packages
- [ ] `make lint` runs successfully
- [ ] README.md with basic project information
