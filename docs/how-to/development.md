# Development Guide

This guide covers the development workflow for ProofBox, a Go implementation of the Jellyfish Merkle Tree.

## Prerequisites

- Go 1.24.2 or later
- golangci-lint (for linting)
- goimports (for import formatting)

## Getting Started

1. **Clone the repository**
   ```bash
   git clone https://github.com/neutral/proofbox.git
   cd proofbox
   ```

2. **Install development tools**
   ```bash
   make install-tools
   ```

3. **Build the project**
   ```bash
   make build
   ```

## Development Workflow

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run benchmarks
make bench

# Run specific package tests
go test -v ./pkg/tree/...
```

### Code Quality

```bash
# Format code
make fmt

# Run linters
make lint

# Run everything (fmt, test, lint)
make all
```

### Building

```bash
# Build the binary
make build

# Clean build artifacts
make clean
```

## Project Structure

```
.
├── cmd/proofbox/    # CLI application
├── pkg/             # Public packages
│   ├── crypto/      # Cryptographic functions
│   ├── proof/       # Proof generation/verification
│   ├── storage/     # Storage abstraction
│   ├── tree/        # Core tree implementation
│   └── types/       # Common types
├── internal/        # Internal packages
├── docs/            # Additional documentation
└── blueprint/       # Design specifications
```

## Coding Standards

### Package Guidelines

- **pkg/**: Contains public packages that can be imported by external projects
- **internal/**: Contains packages that should only be used within ProofBox
- Each package should have a clear, single responsibility
- Package names should be short, concise, and lowercase

### Code Style

- Follow standard Go conventions and idioms
- Use `gofmt` and `goimports` for formatting (automated via `make fmt`)
- Write clear, self-documenting code with meaningful variable names
- Add comments for all exported types, functions, and packages

### Testing

- Write unit tests for all new functionality
- Place tests in `*_test.go` files in the same package
- Use table-driven tests where appropriate
- Aim for >80% code coverage on new code
- Include benchmarks for performance-critical code

### Error Handling

- Always check and handle errors appropriately
- Use descriptive error messages
- Wrap errors with context when propagating them up
- Define custom error types in `pkg/types/errors.go` when needed

## Git Workflow

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**
   - Follow the coding standards
   - Write tests for new functionality
   - Update documentation as needed

3. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

   Follow conventional commit format:
   - `feat:` New feature
   - `fix:` Bug fix
   - `docs:` Documentation changes
   - `test:` Test changes
   - `refactor:` Code refactoring
   - `chore:` Maintenance tasks

4. **Run pre-commit checks**
   ```bash
   make all
   ```

5. **Push and create a pull request**
   ```bash
   git push origin feature/your-feature-name
   ```

## Debugging

### Running with Delve

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug the CLI
dlv debug cmd/proofbox/main.go

# Debug tests
dlv test ./pkg/tree
```

### Profiling

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=. ./pkg/tree
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=. ./pkg/tree
go tool pprof mem.prof
```

## CI/CD

The project uses GitHub Actions for continuous integration. The CI pipeline:

1. Runs on every push and pull request
2. Tests across multiple Go versions (1.21-1.24)
3. Runs linting with golangci-lint
4. Generates code coverage reports

See `.github/workflows/ci.yml` for details.

## Documentation

- **Code Documentation**: Use godoc-style comments for all exported symbols
- **Design Documentation**: Update relevant files in `blueprint/` for design changes
- **API Documentation**: Will be auto-generated from godoc comments
- **User Documentation**: Update README.md for user-facing changes

## Getting Help

- Check existing issues on GitHub
- Consult the blueprint documentation in `blueprint/`
- Review the codebase documentation files (`*.desc.md`)

## License

This project is dual-licensed under CC0-1.0 OR 0BSD. See LICENSE files for details.