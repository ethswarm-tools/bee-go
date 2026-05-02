// Print the chain state Bee currently sees. Read-only.
//
// Usage:
//
//	go run ./examples/chain-state
//
// Environment:
//   - BEE_URL — base URL (default http://localhost:1633)
package main

import (
	"context"
	"fmt"
	"os"

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
	url := getenv("BEE_URL", "http://localhost:1633")
	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	s, err := client.Debug.ChainState(context.Background())
	if err != nil {
		return fmt.Errorf("chain_state: %w", err)
	}

	totalAmount := "0"
	if s.TotalAmount != nil {
		totalAmount = s.TotalAmount.String()
	}
	totalBzz := swarm.NewBZZ(s.TotalAmount)

	lag := uint64(0)
	if s.ChainTip > s.Block {
		lag = s.ChainTip - s.Block
	}

	fmt.Println("Chain state")
	fmt.Println("===========")
	fmt.Printf("Settled block:    %d\n", s.Block)
	fmt.Printf("Chain tip:        %d\n", s.ChainTip)
	fmt.Printf("Lag:              %d block(s)\n", lag)
	fmt.Printf("Current price:    %d PLUR/chunk/block\n", s.CurrentPrice)
	fmt.Printf("Total amount:     %s PLUR\n", totalAmount)
	fmt.Printf("                  %s BZZ\n", totalBzz.ToDecimalString())
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
