package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/neutral/proofbox/pkg/fuzz/corpus"
)

func main() {
	var corpusDir string
	flag.StringVar(&corpusDir, "dir", "pkg/fuzz/corpus/testdata", "Directory to save corpus files (gitignored)")
	flag.Parse()
	
	// Create corpus builder
	builder := corpus.NewCorpusBuilder(corpusDir)
	
	// Build various test cases
	fmt.Println("Building edge cases...")
	builder.BuildEdgeCases()
	
	fmt.Println("Building version isolation cases...")
	builder.BuildVersionIsolationCases()
	
	fmt.Println("Building performance cases...")
	builder.BuildPerformanceCases()
	
	fmt.Println("Building security cases...")
	builder.BuildSecurityCases()
	
	// Save corpus
	fmt.Printf("Saving corpus to %s...\n", corpusDir)
	if err := builder.Save(); err != nil {
		log.Fatalf("Failed to save corpus: %v", err)
	}
	
	fmt.Println("Corpus built successfully!")
	
	// Load and verify
	entries, err := corpus.LoadCorpus(corpusDir)
	if err != nil {
		log.Fatalf("Failed to load corpus: %v", err)
	}
	
	fmt.Printf("\nCorpus contains %d entries:\n", len(entries))
	for _, entry := range entries {
		fmt.Printf("  - %s: %s (tags: %v)\n", entry.Name, entry.Description, entry.Tags)
	}
}