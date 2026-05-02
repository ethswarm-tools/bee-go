// Upload bytes with encrypt=true, observe the 64-byte reference (32
// bytes content address + 32 bytes encryption key), and round-trip
// the data.
//
// Usage:
//
//	go run ./examples/encrypted-upload
//
// Environment:
//   - BEE_URL      — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID — usable postage batch (required)
package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

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

	payload := []byte("some sensitive payload")

	plain, err := client.File.UploadData(ctx, batchID, bytes.NewReader(payload), nil)
	if err != nil {
		return fmt.Errorf("plain upload: %w", err)
	}
	fmt.Printf("plain reference:     %s (%d bytes)\n", plain.Reference.Hex(), plain.Reference.Len())

	yes := true
	enc, err := client.File.UploadData(ctx, batchID, bytes.NewReader(payload), &api.RedundantUploadOptions{
		UploadOptions: api.UploadOptions{Encrypt: &yes},
	})
	if err != nil {
		return fmt.Errorf("encrypted upload: %w", err)
	}
	fmt.Printf("encrypted reference: %s (%d bytes)\n", enc.Reference.Hex(), enc.Reference.Len())

	body, err := client.File.DownloadData(ctx, enc.Reference, nil)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer body.Close()
	got, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if bytes.Equal(got, payload) {
		fmt.Printf("round-trip ok (%d bytes match)\n", len(got))
	} else {
		return fmt.Errorf("round-trip mismatch: got %d bytes, expected %d", len(got), len(payload))
	}
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
