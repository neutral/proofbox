//go:build tools
// +build tools

// Package tools imports various tools used for development.
// This file is not compiled, but `go mod` will track the dependencies.
package tools

import (
	_ "golang.org/x/tools/cmd/goimports"
)