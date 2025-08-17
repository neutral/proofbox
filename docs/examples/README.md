# ProofBox Examples

This directory contains interactive examples demonstrating various features of the ProofBox Jellyfish Merkle Tree implementation. Each example is a standalone Go program that you can run to see the features in action.

## Quick Start

To run any example:

```bash
# From the docs/examples directory
cd batch_basic && go run .

# Or use the Makefile from the docs/examples directory
make                          # Run all examples
make run-batch_basic         # Run a specific example

# Or use the shell scripts
./run_all.sh                 # Run all examples
./run_example.sh batch_basic # Run a specific example
```

## Integration with Main Module

These examples are part of the main ProofBox module and don't have a separate `go.mod`. This ensures they always use the latest local code and makes development easier.

### IDE Configuration

Since these are `package main` programs with shared utilities, some IDEs might show errors. To resolve:

1. **VS Code**: Open the project from the root directory, not the examples directory
2. **GoLand**: Mark the examples directory as "Excluded" if you see errors
3. **Command Line**: Examples will always work with `go run` regardless of IDE errors

The examples are designed to be run, not imported, so IDE warnings can be safely ignored.

## Available Examples

### Batch Operations Examples

- **Basic Operations** (`example_batch_basic.go`) - Fundamental batch transactions
- **Deduplication** (`example_batch_deduplication.go`) - Automatic operation deduplication
- **Parallel Processing** (`example_batch_parallel.go`) - Performance with parallel processing
- **Optimization** (`example_batch_optimization.go`) - Compression and optimization features
- **Concurrent Operations** (`example_batch_concurrent.go`) - Thread-safe concurrent batches
- **Validation** (`example_batch_validation.go`) - Error handling and validation
- **Internals** (`example_batch_internals.go`) - UpdateBatch structure exploration

## Directory Structure

Each example is now in its own directory to avoid multiple main functions in the same package:

```
examples/
├── README.md                   # Main examples documentation
├── Makefile                    # Make targets for running examples
├── run_all.sh                  # Run all examples script
├── run_example.sh              # Run single example script
├── utils/
│   └── utils.go               # Shared utilities package
├── batch_basic/
│   └── main.go                # Basic batch operations
├── batch_concurrent/
│   └── main.go                # Concurrent operations
├── batch_deduplication/
│   └── main.go                # Deduplication demo
├── batch_internals/
│   └── main.go                # Internal structures
├── batch_optimization/
│   └── main.go                # Optimization features
├── batch_parallel/
│   └── main.go                # Parallel processing
└── batch_validation/
    └── main.go                # Validation and errors
```

### Why Separate Directories?

Go requires that all .go files in a directory with `package main` must be compiled together. Having multiple files with `func main()` in the same directory causes compilation errors. By organizing each example in its own directory, we can run them independently while maintaining clean separation.

### Shared Utilities

All examples use a shared `utils` package located in the `utils/` directory. This avoids code duplication while providing consistent output formatting and helper functions across all examples.

### Naming Convention

Examples follow the pattern: `example_<category>_<name>.go`

- `category`: The feature area (e.g., `batch`, `proof`, `storage`)
- `name`: Specific aspect being demonstrated

This allows for logical grouping while maintaining the flat structure.

## Understanding the Output

Each example uses color-coded output for clarity:
- 🔵 **Blue**: Section headers
- 🔷 **Cyan**: Subsection headers
- ✅ **Green**: Success messages
- ❌ **Red**: Error messages
- ➡️ **Yellow**: Information messages
- 🟣 **Purple**: Batch operations
- ⚪ **Gray**: Supplementary information

## Running Examples

There are several ways to run examples:

1. **Run all examples**: `./run_all.sh`
2. **Run specific example with script**: `./run_example.sh batch_basic`
3. **Run directly with go**: `cd batch_basic && go run .`
4. **Use the Makefile**: `make run-batch_basic`

## Adding New Examples

When adding a new example:

1. Create a new directory: `mkdir <category>_<name>`
2. Create `main.go` in the new directory
3. Import `"github.com/neutral/proofbox/docs/examples/utils"` in your main.go
4. Use `utils.PrintSection()`, `utils.CreateTempStorage()`, etc. for consistent output
5. Update the EXAMPLES variable in the Makefile
6. Update this README with the new example
7. Test the example independently and with the run_all script

## Example Structure

Each example follows this general structure:

```go
package main

import (
    "github.com/neutral/proofbox/docs/examples/utils"
    "github.com/neutral/proofbox/pkg/tree"
    // other imports...
)

func main() {
    // Create temporary database
    store, cleanup, err := utils.CreateTempStorage("example-name")
    if err != nil {
        // handle error
    }
    defer cleanup()
    
    // Create tree
    jmt, err := utils.CreateExampleTree(store)
    
    // Demonstrate features
    utils.PrintSection("Example Title")
    // ... example code ...
}
```

## Tips for Exploration

1. **Modify examples** - Change batch sizes, key patterns, or operations to see different behaviors
2. **Enable debug output** - Some examples have verbose modes you can enable
3. **Check temporary databases** - Look in `/tmp/proofbox-example-*` to see the created databases
4. **Compare performance** - Run examples multiple times to see consistency
5. **Mix and match** - Combine concepts from different examples in your own code

## Future Categories

As ProofBox grows, new example categories can be added:
- `proof` - Merkle proof generation and verification
- `storage` - Direct storage operations
- `versioning` - Version management features
- `performance` - Benchmarking and optimization

## Next Steps

After exploring these examples:
- Review the [pkg/tree](../../pkg/tree) package documentation for API details
- Check the [steps](../../steps) directory for implementation progress
- See [docs/](../) for architectural documentation
- Run the test suite to see more usage patterns