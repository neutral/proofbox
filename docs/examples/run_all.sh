#!/bin/bash

# Run all ProofBox examples

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/../.." && pwd )"

echo "🚀 Running ProofBox Examples"
echo "============================"

# Get all example directories
examples=()
for dir in "$SCRIPT_DIR"/*/; do
    if [[ -d "$dir" && -f "$dir/main.go" && "$dir" != *"/metrics/" ]]; then
        examples+=("$dir")
    fi
done

if [ ${#examples[@]} -eq 0 ]; then
    echo "No examples found"
    exit 1
fi

# Sort the examples array
IFS=$'\n' examples=($(sort <<<"${examples[*]}"))
unset IFS

# Run each example
for example_dir in "${examples[@]}"; do
    example_name=$(basename "$example_dir")
    echo ""
    echo "📘 Running $example_name..."
    echo "-----------------------------------"
    
    # Run from the example directory
    cd "$example_dir"
    if go run .; then
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
echo "  cd docs/examples/<name> && go run ."
echo ""
echo "See docs/examples/README.md for more details."