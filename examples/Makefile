# Makefile for ProofBox examples

.PHONY: all clean run-all

# List of all examples
EXAMPLES := batch_basic batch_concurrent batch_deduplication batch_internals batch_optimization batch_parallel batch_validation

# Default target
all: run-all

# Run all examples
run-all:
	@echo "🚀 Running all ProofBox examples..."
	@for example in $(EXAMPLES); do \
		echo "\n📘 Running example_$$example..."; \
		echo "-----------------------------------"; \
		go run example_$$example.go examples_utils.go || echo "❌ example_$$example failed"; \
		echo ""; \
	done

# Run a specific example
run-%:
	@echo "📘 Running example_$*..."
	@go run example_$*.go examples_utils.go

# Build all examples (for testing)
build-all:
	@echo "🔨 Building all examples..."
	@for example in $(EXAMPLES); do \
		echo "Building example_$$example..."; \
		go build -o /tmp/example_$$example example_$$example.go examples_utils.go || exit 1; \
	done
	@echo "✅ All examples built successfully"
	@rm -f /tmp/example_*

# Clean any accidentally compiled binaries
clean:
	@echo "🧹 Cleaning compiled binaries..."
	@rm -f example_batch_*
	@find . -type f -perm +111 -name "example_*" -exec rm {} \;

# Help
help:
	@echo "ProofBox Examples Makefile"
	@echo "========================="
	@echo ""
	@echo "Available targets:"
	@echo "  make              - Run all examples"
	@echo "  make run-all      - Run all examples"
	@echo "  make run-<name>   - Run specific example (e.g., make run-batch_basic)"
	@echo "  make build-all    - Build all examples to test compilation"
	@echo "  make clean        - Clean compiled binaries"
	@echo "  make help         - Show this help"
	@echo ""
	@echo "Available examples:"
	@for example in $(EXAMPLES); do \
		echo "  - $$example"; \
	done