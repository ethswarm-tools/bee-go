// Track an upload's network-sync progress with a Swarm tag.
// Demonstrates the Swarm-Tag header pattern.
//
// Usage:
//
//	go run ./examples/tag-upload-progress
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
	"time"

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
	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}
	ctx := context.Background()

	tag, err := client.API.CreateTag(ctx)
	if err != nil {
		return fmt.Errorf("create_tag: %w", err)
	}
	fmt.Printf("created tag uid=%d\n", tag.UID)

	deferred := true
	payload := bytes.Repeat([]byte{0xc4}, 1024*1024) // 1 MB
	result, err := client.File.UploadData(ctx, batchID, bytes.NewReader(payload), &api.RedundantUploadOptions{
		UploadOptions: api.UploadOptions{
			Tag:      tag.UID,
			Deferred: &deferred,
		},
	})
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	fmt.Printf("upload accepted (%d bytes) → %s\n", len(payload), result.Reference.Hex())

	fmt.Println("\npolling tag every 2s for 10s:")
	fmt.Printf("  %5s  %6s  %6s  %6s  %6s\n", "split", "seen", "stored", "sent", "synced")
	for i := 0; i < 5; i++ {
		t, err := client.API.GetTag(ctx, tag.UID)
		if err != nil {
			return fmt.Errorf("get_tag: %w", err)
		}
		fmt.Printf("  %5d  %6d  %6d  %6d  %6d\n", t.Split, t.Seen, t.Stored, t.Sent, t.Synced)
		time.Sleep(2 * time.Second)
	}
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
