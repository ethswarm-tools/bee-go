// Full pin lifecycle: upload → pin → list → is_retrievable →
// reupload → unpin → re-pin.
//
// Usage:
//
//	go run ./examples/pinning-workflow
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

	// 1. Upload bytes (no pin flag yet).
	payload := []byte("hello pinning workflow")
	result, err := client.File.UploadData(ctx, batchID, bytes.NewReader(payload), nil)
	if err != nil {
		return fmt.Errorf("upload_data: %w", err)
	}
	ref := result.Reference
	fmt.Printf("1. uploaded → %s\n", ref.Hex())

	// 2. Pin.
	if err := client.API.Pin(ctx, ref); err != nil {
		return fmt.Errorf("pin: %w", err)
	}
	fmt.Println("2. pinned")

	// 3. Confirm via get_pin and list_pins.
	pinned, err := client.API.GetPin(ctx, ref)
	if err != nil {
		return fmt.Errorf("get_pin: %w", err)
	}
	pins, err := client.API.ListPins(ctx)
	if err != nil {
		return fmt.Errorf("list_pins: %w", err)
	}
	found := false
	for _, p := range pins {
		if p.Hex() == ref.Hex() {
			found = true
			break
		}
	}
	fmt.Printf("3. get_pin = %t; list_pins has %d pin(s) (this one included: %t)\n", pinned, len(pins), found)

	// 4. Stewardship: is reference retrievable?
	retrievable, err := client.API.IsRetrievable(ctx, ref)
	if err != nil {
		return fmt.Errorf("is_retrievable: %w", err)
	}
	fmt.Printf("4. is_retrievable = %t\n", retrievable)

	// 5. Reupload pinned data.
	if err := client.API.Reupload(ctx, ref, batchID); err != nil {
		return fmt.Errorf("reupload: %w", err)
	}
	fmt.Println("5. reuploaded pinned data")

	// 6. Unpin.
	if err := client.API.Unpin(ctx, ref); err != nil {
		return fmt.Errorf("unpin: %w", err)
	}
	after, err := client.API.GetPin(ctx, ref)
	if err != nil {
		return fmt.Errorf("get_pin after unpin: %w", err)
	}
	fmt.Printf("6. unpinned; get_pin now = %t\n", after)

	// 7. Re-pin.
	if err := client.API.Pin(ctx, ref); err != nil {
		return fmt.Errorf("re-pin: %w", err)
	}
	fmt.Println("7. re-pinned")
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
