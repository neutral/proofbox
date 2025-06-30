# ProofBox Examples

This directory contains interactive examples demonstrating various features of the ProofBox Jellyfish Merkle Tree implementation. Each example is a standalone Go program that you can run to see the features in action.

## Quick Start

To run any example:

```bash
# From the project root
cd examples && go run example_batch_basic.go examples_utils.go

# Or use the convenience scripts
./examples/run_all.sh              # Run all examples
./examples/run_example.sh batch_basic  # Run a specific example
```

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

All examples are kept in a flat structure within the `examples/` directory:

```
examples/
├── README.md                    # Main examples documentation
├── examples_utils.go           # Shared utility functions
├── examples_utils.go.desc.md   # Utils documentation
├── go.mod                      # Module definition
├── run_all.sh                  # Run all examples script
├── run_example.sh              # Run single example script
├── example_batch_basic.go      # Basic batch operations
├── example_batch_concurrent.go # Concurrent operations
├── example_batch_deduplication.go # Deduplication demo
├── example_batch_internals.go  # Internal structures
├── example_batch_optimization.go # Optimization features
├── example_batch_parallel.go   # Parallel processing
└── example_batch_validation.go # Validation and errors
```

### Why a Flat Structure?

Go's `go run` command requires all named files to be in the same directory. Since our examples share common utilities (`examples_utils.go`), we keep all examples in one directory to simplify execution.

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

There are three ways to run examples:

1. **Run all examples**: `./run_all.sh`
2. **Run specific example with script**: `./run_example.sh batch_basic`
3. **Run directly with go**: `go run example_batch_basic.go examples_utils.go`

## Adding New Examples

When adding a new example:

1. Create `example_<category>_<name>.go` in the examples directory
2. Import required packages (no need to import utils - they're in the same package)
3. Use the utility functions for consistent output
4. Add run instructions at the bottom
5. Update this README with the new example
6. Test the example independently and with the run_all script

## Example Structure

Each example follows this general structure:

```go
package main

import (
    "github.com/neutral/proofbox/pkg/tree"
    // other imports...
)

func main() {
    // Create temporary database
    db, cleanup, err := CreateTempDB("example-name")
    if err != nil {
        // handle error
    }
    defer cleanup()
    
    // Create tree
    jmt, err := tree.NewTree(db, tree.DefaultTreeConfig())
    
    // Demonstrate features
    PrintSection("Example Title")
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
- Review the [pkg/tree](../pkg/tree) package documentation for API details
- Check the [blueprint/steps](../blueprint/steps) directory for implementation progress
- See [docs/](../docs/) for architectural documentation
- Run the test suite to see more usage patterns