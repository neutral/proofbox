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
		echo "\n📘 Running $$example..."; \
		echo "-----------------------------------"; \
		cd $$example && go run . || echo "❌ $$example failed"; \
		cd ..; \
		echo ""; \
	done

# Run a specific example
run-%:
	@echo "📘 Running $*..."
	@cd $* && go run .

# Build all examples (for testing)
build-all:
	@echo "🔨 Building all examples..."
	@for example in $(EXAMPLES); do \
		echo "Building $$example..."; \
		cd $$example && go build -o /tmp/$$example . || exit 1; \
		cd ..; \
	done
	@echo "✅ All examples built successfully"
	@rm -f /tmp/batch_*

# Clean any accidentally compiled binaries
clean:
	@echo "🧹 Cleaning compiled binaries..."
	@for example in $(EXAMPLES); do \
		rm -f $$example/$$example; \
		find $$example -type f -perm +111 -not -name "*.sh" -exec rm {} \; 2>/dev/null || true; \
	done

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