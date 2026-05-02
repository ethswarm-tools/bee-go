// List every postage batch owned by this Bee node. Read-only.
//
// Usage:
//
//	go run ./examples/list-batches
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
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	url := getenv("BEE_URL", "http://localhost:1633")
	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}

	batches, err := client.Postage.GetPostageBatches(context.Background())
	if err != nil {
		return fmt.Errorf("get_postage_batches: %w", err)
	}
	if len(batches) == 0 {
		fmt.Println("No postage batches owned by this node.")
		return nil
	}

	fmt.Printf("%d batch(es):\n\n", len(batches))
	fmt.Printf("%-64s  %5s  %9s  %6s  %11s  %9s  %-8s  %s\n",
		"batch id", "depth", "amount", "usable", "ttl(s)", "util(%)", "immut", "label")
	fmt.Println(strings.Repeat("-", 140))
	for _, b := range batches {
		amount := "-"
		if b.Amount != nil {
			amount = b.Amount.String()
		}
		var utilPct float64
		if b.Depth > b.BucketDepth {
			cap := uint64(1) << (b.Depth - b.BucketDepth)
			utilPct = float64(b.Utilization) / float64(cap) * 100.0
		}
		fmt.Printf("%-64s  %5d  %9s  %6t  %11d  %8.2f%%  %-8t  %s\n",
			b.BatchID.Hex(), b.Depth, amount, b.Usable, b.BatchTTL, utilPct, b.Immutable, b.Label)
	}
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
