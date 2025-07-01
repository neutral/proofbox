#!/bin/bash

# Simple script to run a single example

if [ $# -eq 0 ]; then
    echo "Usage: ./run_example.sh <example_name>"
    echo "Example: ./run_example.sh batch_basic"
    exit 1
fi

EXAMPLE_NAME=$1
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Remove any path components if provided
EXAMPLE_NAME=$(basename "$EXAMPLE_NAME")

# Check if directory exists
if [ ! -d "$SCRIPT_DIR/$EXAMPLE_NAME" ]; then
    echo "Error: Example directory '$EXAMPLE_NAME' not found"
    echo "Available examples:"
    for dir in "$SCRIPT_DIR"/*/; do
        if [[ -d "$dir" && -f "$dir/main.go" && "$dir" != *"/metrics/" ]]; then
            echo "  - $(basename "$dir")"
        fi
    done
    exit 1
fi

# Check if main.go exists in the directory
if [ ! -f "$SCRIPT_DIR/$EXAMPLE_NAME/main.go" ]; then
    echo "Error: main.go not found in $EXAMPLE_NAME directory"
    exit 1
fi

# Run the example
echo "Running $EXAMPLE_NAME..."
cd "$SCRIPT_DIR/$EXAMPLE_NAME" && go run .