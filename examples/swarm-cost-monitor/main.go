// swarm-cost-monitor is an operator dashboard: batch TTLs counting
// down, current chain price, projected refill cost.
//
// Usage:
//
//	swarm-cost-monitor                 # one-shot snapshot
//	swarm-cost-monitor watch           # refresh every 30s
//	swarm-cost-monitor refill --days 30
//
// Environment:
//   - BEE_URL           — base URL (default http://localhost:1633)
//   - BEE_BLOCK_SECONDS — chain block time (default 5 Gnosis,
//     set to 15 for mainnet-like chains)
package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"time"

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
	url := getenv("BEE_URL", "http://localhost:1633")
	args := os.Args[1:]
	cmd := "snapshot"
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}
	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	switch cmd {
	case "snapshot":
		return snapshot(client)
	case "watch":
		return watch(client)
	case "refill":
		days := 30.0
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--days":
				i++
				if i >= len(args) {
					return fmt.Errorf("--days needs N")
				}
				n, err := strconv.ParseFloat(args[i], 64)
				if err != nil {
					return fmt.Errorf("invalid days: %w", err)
				}
				days = n
			default:
				return fmt.Errorf("unknown flag: %s", args[i])
			}
		}
		return refill(client, days)
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func snapshot(client *bee.Client) error {
	batches, err := client.Postage.GetPostageBatches(context.Background())
	if err != nil {
		return fmt.Errorf("get_postage_batches: %w", err)
	}
	chain, err := client.Debug.ChainState(context.Background())
	if err != nil {
		return fmt.Errorf("chain_state: %w", err)
	}
	blockSecs := blockSeconds()

	fmt.Printf("chain: block=%d tip=%d price=%d PLUR/chunk/block\n",
		chain.Block, chain.ChainTip, chain.CurrentPrice)
	_ = blockSecs
	if chain.ChainTip > chain.Block {
		fmt.Printf("       (Bee is %d blocks behind tip)\n", chain.ChainTip-chain.Block)
	}

	if len(batches) == 0 {
		fmt.Println("\n(no batches owned by this node)")
		return nil
	}
	fmt.Printf("\n%-10s  %-6s  %-10s  %-14s  %-14s  %s\n",
		"id8", "depth", "usable", "ttl", "fill", "label")
	for _, b := range batches {
		id8 := b.BatchID.Hex()[:8]
		usage := postage.GetStampUsage(b.Utilization, b.Depth, b.BucketDepth)
		ttl := formatTTL(b.BatchTTL)
		fmt.Printf("%-10s  %-6d  %-10s  %-14s  %-13.1f%%  %s\n",
			id8, b.Depth, yesNo(b.Usable), ttl, usage*100, b.Label)
	}
	fmt.Println()
	showWarnings(batches, blockSecs)
	return nil
}

func watch(client *bee.Client) error {
	for {
		fmt.Print("\033[2J\033[H")
		if err := snapshot(client); err != nil {
			return err
		}
		time.Sleep(30 * time.Second)
	}
}

func refill(client *bee.Client, days float64) error {
	chain, err := client.Debug.ChainState(context.Background())
	if err != nil {
		return fmt.Errorf("chain_state: %w", err)
	}
	blockSecs := blockSeconds()
	blocks := uint64(days * 86400.0 / float64(blockSecs))
	batches, err := client.Postage.GetPostageBatches(context.Background())
	if err != nil {
		return fmt.Errorf("get_postage_batches: %w", err)
	}
	fmt.Printf("Refill projection (%.1fd at %d PLUR/chunk/block, %ds blocks):\n",
		days, chain.CurrentPrice, blockSecs)
	if len(batches) == 0 {
		fmt.Println("(no batches owned)")
		return nil
	}
	fmt.Printf("\n%-10s  %-6s  %-14s  %-22s  refill cost (BZZ)\n",
		"id8", "depth", "current ttl", "topup amount/chunk")
	total := big.NewInt(0)
	for _, b := range batches {
		id8 := b.BatchID.Hex()[:8]
		topupPerChunk := new(big.Int).Mul(big.NewInt(int64(chain.CurrentPrice)), big.NewInt(int64(blocks)))
		topupTotal := postage.GetStampCost(int(b.Depth), topupPerChunk)
		bzz := swarm.NewBZZ(topupTotal)
		total = new(big.Int).Add(total, topupTotal)
		fmt.Printf("%-10s  %-6d  %-14s  %-22s  %s\n",
			id8, b.Depth, formatTTL(b.BatchTTL), topupPerChunk.String(),
			bzz.ToSignificantDigits(4))
	}
	totalBZZ := swarm.NewBZZ(total)
	fmt.Printf("\nTotal projected refill: %s BZZ\n", totalBZZ.ToSignificantDigits(4))
	return nil
}

func showWarnings(batches []postage.PostageBatch, blockSecs uint64) {
	warned := false
	for _, b := range batches {
		if b.BatchTTL > 0 && b.BatchTTL < 7*86400 {
			warned = true
			days := float64(b.BatchTTL) / 86400.0
			fmt.Printf("WARN batch %s TTL %.1fd — refill soon\n", b.BatchID.Hex()[:8], days)
		}
		usage := postage.GetStampUsage(b.Utilization, b.Depth, b.BucketDepth)
		if usage > 0.85 {
			warned = true
			fmt.Printf("WARN batch %s %.1f%% full — dilute soon\n", b.BatchID.Hex()[:8], usage*100)
		}
	}
	if !warned {
		_ = blockSecs
		fmt.Println("(all batches healthy)")
	}
}

func formatTTL(secs int64) string {
	if secs < 0 {
		return "n/a"
	}
	s := uint64(secs)
	days := s / 86400
	hours := (s % 86400) / 3600
	if days > 0 {
		return fmt.Sprintf("%dd%02dh", days, hours)
	}
	return fmt.Sprintf("%dh%02dm", s/3600, (s%3600)/60)
}

func blockSeconds() uint64 {
	if v := os.Getenv("BEE_BLOCK_SECONDS"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			return n
		}
	}
	return 5
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
