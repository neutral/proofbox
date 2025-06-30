#!/bin/bash

# Run all ProofBox examples

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

echo "🚀 Running ProofBox Examples"
echo "============================"

# Get all example files
examples=()
while IFS= read -r -d '' file; do
    examples+=("$file")
done < <(find "$SCRIPT_DIR" -name "example_*.go" -print0 | sort -z)

if [ ${#examples[@]} -eq 0 ]; then
    echo "No examples found"
    exit 1
fi

# Run each example
for example in "${examples[@]}"; do
    example_name=$(basename "$example" .go)
    echo ""
    echo "📘 Running $example_name..."
    echo "-----------------------------------"
    
    # Run from examples directory
    cd "$SCRIPT_DIR"
    if go run "$(basename "$example")" examples_utils.go; then
        echo "✅ $example_name completed successfully"
    else
        echo "❌ $example_name failed"
    fi
    
    echo ""
    echo "Press Enter to continue to the next example..."
    read -r
done

echo ""
echo "🎉 All examples completed!"
echo ""
echo "To run individual examples:"
echo "  cd examples && go run example_<name>.go examples_utils.go"
echo ""
echo "See examples/README.md for more details."