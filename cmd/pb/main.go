// Package main provides the ProofBox CLI tool
package main

import (
	"os"

	"github.com/neutral/proofbox/cmd/pb/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
