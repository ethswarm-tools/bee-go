// Per-bucket fill analysis for one batch. Predicts when the batch
// will be full. Read-only.
//
// Usage:
//
//	go run ./examples/stamp-utilization <batch-id>
//
// Environment:
//   - BEE_URL — base URL (default http://localhost:1633)
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: stamp-utilization <batch-id>")
	}
	url := getenv("BEE_URL", "http://localhost:1633")
	batchID, err := swarm.BatchIDFromHex(os.Args[1])
	if err != nil {
		return fmt.Errorf("invalid batch id: %w", err)
	}

	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	buckets, err := client.Postage.GetPostageBatchBuckets(context.Background(), batchID)
	if err != nil {
		return fmt.Errorf("get_postage_batch_buckets: %w", err)
	}

	totalBuckets := len(buckets.Buckets)
	var maxFill, totalChunks uint64
	var filledBuckets int
	for _, b := range buckets.Buckets {
		if b.Collisions > 0 {
			filledBuckets++
		}
		if uint64(b.Collisions) > maxFill {
			maxFill = uint64(b.Collisions)
		}
		totalChunks += uint64(b.Collisions)
	}
	cap := buckets.BucketUpperBound
	var pctFullBuckets, maxFillPct float64
	if totalBuckets > 0 {
		pctFullBuckets = float64(filledBuckets) / float64(totalBuckets) * 100.0
	}
	if cap > 0 {
		maxFillPct = float64(maxFill) / float64(cap) * 100.0
	}

	fmt.Printf("Stamp utilization for batch %s\n", batchID.Hex())
	fmt.Println(strings.Repeat("=", 65))
	fmt.Printf("Depth:                 %d\n", buckets.Depth)
	fmt.Printf("Bucket depth:          %d\n", buckets.BucketDepth)
	fmt.Printf("Per-bucket cap:        %d\n", cap)
	fmt.Printf("Total buckets:         %d\n", totalBuckets)
	fmt.Printf("Buckets used:          %d (%.2f%%)\n", filledBuckets, pctFullBuckets)
	fmt.Printf("Hottest bucket fill:   %d / %d (%.2f%%)\n", maxFill, cap, maxFillPct)
	fmt.Printf("Total chunks stamped:  %d\n", totalChunks)
	fmt.Println()
	fmt.Println("The hottest bucket determines when the batch becomes full —")
	fmt.Println("Bee rejects writes once any bucket hits the cap. A high")
	fmt.Println("imbalance is a sign you should dilute (deepen) the batch.")
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
