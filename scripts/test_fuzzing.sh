#!/bin/bash

# Test script to verify fuzzing setup works locally and for CI

set -e

echo "========================================="
echo "Testing Fuzzing Setup"
echo "========================================="

echo ""
echo "1. Testing quick property-based tests (rapid)..."
make fuzz-quick
echo "✓ Quick property tests passed"

echo ""
echo "2. Testing native Go fuzzing - FuzzTreeOperations..."
make fuzz FUZZ=FuzzTreeOperations FUZZTIME=5s
echo "✓ Native fuzzing FuzzTreeOperations passed"

echo ""
echo "3. Testing native Go fuzzing - FuzzProofGeneration..."
make fuzz FUZZ=FuzzProofGeneration FUZZTIME=5s
echo "✓ Native fuzzing FuzzProofGeneration passed"

echo ""
echo "4. Testing corpus builder (if exists)..."
if [ -f pkg/fuzz/corpus/cmd/build_corpus.go ]; then
  mkdir -p testdata/fuzz/corpus
  go run pkg/fuzz/corpus/cmd/build_corpus.go -dir testdata/fuzz/corpus
  echo "✓ Corpus builder executed successfully"
else
  echo "⚠ Corpus builder not found (expected for minimal setup)"
fi

echo ""
echo "========================================="
echo "All fuzzing tests completed successfully!"
echo "========================================="