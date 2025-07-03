# Test Configuration

This file provides centralized configuration for fuzzing test iteration counts and levels.

## Purpose

The test configuration system allows fuzzing tests to adapt their behavior based on the testing context (quick PR checks, standard tests, or comprehensive nightly runs). This ensures appropriate test coverage while maintaining fast feedback loops during development.

## Environment Variables

- `FUZZ_ITERATIONS`: Override iteration count with a specific value
- `FUZZ_LEVEL`: Set test level (quick/standard/comprehensive/stress)

## Test Levels

- **quick**: 20 iterations for rapid CI/PR feedback (~30 seconds)
- **standard**: 100 iterations for default testing (~5 minutes)
- **comprehensive**: 1000 iterations for thorough testing (~1 hour)
- **stress**: 10000 iterations for manual stress testing

This configuration implements the testing methodology specified in the blueprint, enabling different levels of test thoroughness based on the development phase.