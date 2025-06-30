#!/bin/bash

# Simple script to run a single example with the utils

if [ $# -eq 0 ]; then
    echo "Usage: ./run_example.sh <example_name>"
    echo "Example: ./run_example.sh batch_basic"
    exit 1
fi

EXAMPLE_NAME=$1

# Remove "example_" prefix if provided
EXAMPLE_NAME=${EXAMPLE_NAME#example_}

# Add .go extension if not provided
if [[ ! "$EXAMPLE_NAME" == *.go ]]; then
    EXAMPLE_NAME="example_${EXAMPLE_NAME}.go"
else
    # If .go was provided, ensure it has example_ prefix
    if [[ ! "$EXAMPLE_NAME" == example_* ]]; then
        EXAMPLE_NAME="example_${EXAMPLE_NAME}"
    fi
fi

# Check if file exists
if [ ! -f "$EXAMPLE_NAME" ]; then
    echo "Error: $EXAMPLE_NAME not found"
    echo "Available examples:"
    ls example_*.go | sed 's/example_/  - /g' | sed 's/.go//g'
    exit 1
fi

# Run the example with utils
echo "Running $EXAMPLE_NAME..."
go run "$EXAMPLE_NAME" examples_utils.go