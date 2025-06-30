# examples_utils.go

This file provides common utilities for all ProofBox examples.

## Purpose

The utilities in this file enable consistent, color-coded terminal output across all examples, making them easier to read and understand. It also provides helper functions for common operations like creating temporary databases and formatting data.

## Key Functions

- **Color constants**: Define ANSI escape codes for terminal colors
- **Print functions**: Consistent output formatting (PrintSection, PrintSuccess, PrintError, etc.)
- **Database helpers**: CreateTempDB for isolated example databases
- **Tree initialization**: InitializeTreeWithData for pre-populated trees
- **Data formatting**: FormatBytes, CompareVersions for readable output
- **Batch utilities**: PrintBatchOperation for visualizing batch operations

## Usage Pattern

Examples should use these utilities to maintain consistency:
- Use PrintSection for major divisions
- Use PrintSubSection for minor divisions
- Use PrintSuccess/PrintError for results
- Use PrintInfo for explanatory text
- Use PrintKeyValue for data display

This ensures all examples have a uniform, professional appearance that helps users understand the functionality being demonstrated.