// Preview the BZZ cost of buying a batch using the live /chainstate
// price. Read-only (no chain TX).
//
// Usage:
//
//	go run ./examples/stamp-cost-live [size] [duration] [network]
//
// Defaults: size=1GB, duration=30d, network=gnosis (5s blocks; pass
// "mainnet" for 15s).
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
	"github.com/ethswarm-tools/bee-go/pkg/postage"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]
	sizeStr := pick(args, 0, "1GB")
	durStr := pick(args, 1, "30d")
	netStr := pick(args, 2, "gnosis")

	url := getenv("BEE_URL", "http://localhost:1633")
	size, err := swarm.SizeFromString(sizeStr)
	if err != nil {
		return fmt.Errorf("invalid size: %w", err)
	}
	duration, err := swarm.DurationFromString(durStr)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	var network bee.Network
	switch strings.ToLower(netStr) {
	case "gnosis":
		network = bee.NetworkGnosis
	case "mainnet":
		network = bee.NetworkMainnet
	default:
		return fmt.Errorf("unknown network %q (gnosis or mainnet)", netStr)
	}

	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	chain, err := client.Debug.ChainState(context.Background())
	if err != nil {
		return fmt.Errorf("chain_state: %w", err)
	}
	bzz, err := client.GetStorageCost(context.Background(), size, duration, &bee.StorageOptions{Network: network})
	if err != nil {
		return fmt.Errorf("get_storage_cost: %w", err)
	}

	depth := postage.GetDepthForSize(size)
	blockTime := network.BlockTimeSeconds()
	blocks := uint64(duration.ToSeconds()) / blockTime

	fmt.Println("Live stamp cost preview")
	fmt.Println("=======================")
	fmt.Printf("Bee URL:              %s\n", url)
	fmt.Printf("Size:                 %s (%d bytes)\n", sizeStr, size.ToBytes())
	fmt.Printf("Duration:             %s\n", durStr)
	fmt.Printf("Network:              %s (%ds blocks)\n", netStr, blockTime)
	fmt.Println()
	fmt.Printf("Live chain price:     %d PLUR/chunk/block\n", chain.CurrentPrice)
	fmt.Printf("Stamp depth:          %d\n", depth)
	fmt.Printf("Blocks for duration:  %d\n", blocks)
	fmt.Printf("Total cost:           %s PLUR\n", bzz.ToPLURString())
	fmt.Printf("                      %s BZZ\n", bzz.ToDecimalString())
	return nil
}

func pick(args []string, i int, def string) string {
	if i < len(args) {
		return args[i]
	}
	return def
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
