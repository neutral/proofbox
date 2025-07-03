package main

import (
	"bytes"
	"context"
	"log"
	"time"

	"github.com/neutral/proofbox/pkg/metrics"
	"github.com/neutral/proofbox/pkg/storage"
	"github.com/neutral/proofbox/pkg/storage/memory"
	"github.com/neutral/proofbox/pkg/tree"
	"github.com/neutral/proofbox/pkg/types"
)

func main() {
	// Create metrics collector
	collector := metrics.NewCollector(true)

	// Start metrics server
	server := metrics.NewMetricsServer(":9090", collector)
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Stop(ctx)
	}()

	log.Printf("Metrics server started on %s", server.GetAddr())
	log.Printf("Visit http://localhost:9090/metrics to see Prometheus metrics")

	// Create storage and tree with metrics
	store := memory.NewStorage()
	defer store.Close()

	keyEncoder := storage.NewDefaultKeyEncoder()
	config := tree.DefaultTreeConfig()
	config.MetricsEnabled = true
	config.Metrics = collector.JMT()

	tr, err := tree.NewTree(store, keyEncoder, config)
	if err != nil {
		log.Fatal(err)
	}

	// Perform operations that generate metrics
	log.Println("Performing operations to generate metrics...")

	for i := 0; i < 100; i++ {
		key := types.KeyHash([]byte("example-key-" + string(rune(i))))
		value := []byte("example-value-" + string(rune(i)))

		// Put operation
		version, err := tr.Put(key, value)
		if err != nil {
			log.Printf("Failed to put key %d: %v", i, err)
			continue
		}

		// Get operation
		retrieved, err := tr.Get(version, key)
		if err != nil {
			log.Printf("Failed to get key %d: %v", i, err)
			continue
		}

		if !bytes.Equal(retrieved, value) {
			log.Printf("Value mismatch for key %d", i)
		}

		// Generate proof
		reader, err := tr.Reader(version)
		if err != nil {
			log.Printf("Failed to create reader for key %d: %v", i, err)
			continue
		}
		reader.Close()

		if i%10 == 0 {
			log.Printf("Completed %d operations", i+1)
		}
	}

	log.Println("Operations completed!")
	log.Println("Check http://localhost:9090/metrics for the collected metrics")
	log.Println("Press Ctrl+C to exit...")

	// Keep running to allow metrics inspection
	select {}
}
