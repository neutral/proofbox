# Development Commands

## Build and Run
```bash
# Build the project
go build

# Run the application
go run main.go

# Build for production
go build -o proofbox
```

## Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detector
go test -race ./...

# Run specific test
go test -run TestName ./...

# Run benchmarks
go test -bench=. ./...
```

## Code Quality
```bash
# Format code
go fmt ./...

# Lint code
go vet ./...

# Run staticcheck (if installed)
staticcheck ./...
```

## Dependencies
```bash
# Download dependencies
go mod download

# Tidy dependencies
go mod tidy

# Verify dependencies
go mod verify
```