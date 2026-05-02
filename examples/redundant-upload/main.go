// Upload data with erasure-coded redundancy. The upload pays for
// extra parity chunks so the data survives even when some hosting
// nodes go offline.
//
// Usage:
//
//	go run ./examples/redundant-upload [level]
//
// `level` is off|medium|strong|insane|paranoid. Defaults to "medium".
//
// Environment:
//   - BEE_URL      — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID — usable postage batch (required)
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/api"
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
	batchHex := os.Getenv("BEE_BATCH_ID")
	if batchHex == "" {
		return fmt.Errorf("BEE_BATCH_ID is required")
	}
	batchID, err := swarm.BatchIDFromHex(batchHex)
	if err != nil {
		return fmt.Errorf("invalid BEE_BATCH_ID: %w", err)
	}

	levelStr := "medium"
	if len(os.Args) >= 2 {
		levelStr = os.Args[1]
	}
	var level api.RedundancyLevel
	switch strings.ToLower(levelStr) {
	case "off":
		level = api.RedundancyLevelOff
	case "medium":
		level = api.RedundancyLevelMedium
	case "strong":
		level = api.RedundancyLevelStrong
	case "insane":
		level = api.RedundancyLevelInsane
	case "paranoid":
		level = api.RedundancyLevelParanoid
	default:
		return fmt.Errorf("unknown level %q (off|medium|strong|insane|paranoid)", levelStr)
	}

	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	ctx := context.Background()

	payload := bytes.Repeat([]byte{0xa5}, 256*1024) // 256 KB

	plain, err := client.File.UploadData(ctx, batchID, bytes.NewReader(payload), nil)
	if err != nil {
		return fmt.Errorf("plain upload: %w", err)
	}
	fmt.Printf("Plain upload (off):       %s\n", plain.Reference.Hex())

	redundant, err := client.File.UploadData(ctx, batchID, bytes.NewReader(payload), &api.RedundantUploadOptions{
		RedundancyLevel: level,
	})
	if err != nil {
		return fmt.Errorf("redundant upload: %w", err)
	}
	fmt.Printf("Redundant upload (%-8s): %s\n", levelStr, redundant.Reference.Hex())
	fmt.Println()
	fmt.Printf("Payload size: %d bytes. Higher redundancy levels stamp more\n", len(payload))
	fmt.Println("parity chunks; the reference is the same 'visible' root.")
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
