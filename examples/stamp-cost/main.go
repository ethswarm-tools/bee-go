// Pure offline calculator: given a size, duration, per-chunk-per-block
// price, and a network, print the postage stamp depth and the total
// BZZ cost. No Bee node required.
//
// Usage:
//
//	go run ./examples/stamp-cost [size] [duration] [price-plur] [network]
//
// Defaults: size=1GB, duration=30d, price=24000 PLUR/chunk/block,
// network=gnosis (5s blocks; use "mainnet" for 15s).
//
// For the live chain price, run a Bee node and use Client.GetStorageCost
// instead — it queries /chainstate.
package main

import (
	"fmt"
	"math/big"
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
	priceStr := pick(args, 2, "24000")
	netStr := pick(args, 3, "gnosis")

	size, err := swarm.SizeFromString(sizeStr)
	if err != nil {
		return fmt.Errorf("invalid size %q: %w", sizeStr, err)
	}
	duration, err := swarm.DurationFromString(durStr)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", durStr, err)
	}
	pricePerBlock, ok := new(big.Int).SetString(priceStr, 10)
	if !ok {
		return fmt.Errorf("invalid price %q (want a base-10 integer in PLUR)", priceStr)
	}
	var network bee.Network
	switch strings.ToLower(netStr) {
	case "gnosis":
		network = bee.NetworkGnosis
	case "mainnet":
		network = bee.NetworkMainnet
	default:
		return fmt.Errorf("unknown network %q (want gnosis or mainnet)", netStr)
	}

	depth := postage.GetDepthForSize(size)
	blockTime := network.BlockTimeSeconds()

	// amount per chunk = pricePerBlock * blocks_in_duration.
	// Computed inline (rather than via postage.GetAmountForDuration) so
	// the math is fully visible.
	blocks := uint64(duration.ToSeconds()) / blockTime
	amountPerChunk := new(big.Int).Mul(pricePerBlock, new(big.Int).SetUint64(blocks))
	totalPLUR := postage.GetStampCost(depth, amountPerChunk)
	totalBZZ := swarm.NewBZZ(totalPLUR)

	chunks := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(depth)), nil)

	fmt.Println("Stamp cost preview")
	fmt.Println("==================")
	fmt.Printf("Size:                  %s (%d bytes)\n", sizeStr, size.ToBytes())
	fmt.Printf("Duration:              %s (%d seconds, ~%.2f days)\n", durStr, duration.ToSeconds(), duration.ToDays())
	fmt.Printf("Network:               %s (%ds blocks)\n", netStr, blockTime)
	fmt.Printf("Price per chunk/block: %s PLUR\n", pricePerBlock.String())
	fmt.Println()
	fmt.Printf("Stamp depth:           %d\n", depth)
	fmt.Printf("Chunks covered:        2^%d = %s\n", depth, chunks.String())
	fmt.Printf("Blocks for duration:   %d\n", blocks)
	fmt.Printf("Per-chunk amount:      %s PLUR\n", amountPerChunk.String())
	fmt.Printf("Total cost:            %s PLUR\n", totalPLUR.String())
	fmt.Printf("                       %s BZZ\n", totalBZZ.ToDecimalString())

	return nil
}

func pick(args []string, i int, def string) string {
	if i < len(args) {
		return args[i]
	}
	return def
}
