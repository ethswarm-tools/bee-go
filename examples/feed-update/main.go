// Sign and publish a Swarm feed update, then read it back.
//
// A Swarm feed is a mutable pointer keyed on (owner, topic). Each
// update is signed by the owner and indexed sequentially; readers
// ask for the latest update without knowing the index ahead of time.
//
// Usage:
//
//	go run ./examples/feed-update <topic-string> <message>
//
// Environment:
//   - BEE_URL         — base URL (default http://localhost:1633)
//   - BEE_BATCH_ID    — usable postage batch hex (required)
//   - BEE_SIGNER_HEX  — 32-byte hex private key (required).
//     Generate one with `openssl rand -hex 32`. Re-using the same
//     signer + topic updates the same feed; a new signer creates a
//     new feed.
package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"time"
	"unicode/utf8"

	bee "github.com/ethswarm-tools/bee-go"
	"github.com/ethswarm-tools/bee-go/pkg/file"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: feed-update <topic-string> <message>")
	}
	topicStr := os.Args[1]
	message := os.Args[2]

	url := getenv("BEE_URL", "http://localhost:1633")
	batchHex := os.Getenv("BEE_BATCH_ID")
	if batchHex == "" {
		return fmt.Errorf("BEE_BATCH_ID is required (set to a usable batch hex id)")
	}
	signerHex := os.Getenv("BEE_SIGNER_HEX")
	if signerHex == "" {
		return fmt.Errorf("BEE_SIGNER_HEX is required (32-byte hex). Generate one with `openssl rand -hex 32`")
	}

	batchID, err := swarm.BatchIDFromHex(batchHex)
	if err != nil {
		return fmt.Errorf("invalid BEE_BATCH_ID: %w", err)
	}
	signer, err := swarm.PrivateKeyFromHex(signerHex)
	if err != nil {
		return fmt.Errorf("invalid BEE_SIGNER_HEX: %w", err)
	}
	owner := signer.PublicKey().Address()
	topic := swarm.TopicFromString(topicStr)

	fmt.Println("Feed parameters:")
	fmt.Printf("- Owner:  %s\n", owner.Hex())
	fmt.Printf("- Topic:  %s (from %q)\n", topic.Hex(), topicStr)
	fmt.Printf("- Batch:  %s\n\n", batchID.Hex())

	client, err := bee.NewClient(url)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	fmt.Printf("Publishing feed update with payload %q...\n", message)
	result, err := client.File.UpdateFeed(context.Background(), batchID, signer, topic, []byte(message))
	if err != nil {
		return fmt.Errorf("update_feed failed: %w", err)
	}
	fmt.Printf("  chunk reference: %s\n", result.Reference.Hex())

	// Bee can take a while to index a freshly-uploaded feed SOC the
	// first time it sees a given (owner, topic). Retry with backoff
	// so the first lookup doesn't 404.
	fmt.Println("\nFetching latest feed update...")
	var update file.FeedUpdate
	for _, delayMs := range []int{500, 1000, 2000, 4000, 8000, 14000, 30000} {
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
		update, err = client.File.FetchLatestFeedUpdate(context.Background(), owner, topic)
		if err == nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("fetch_latest_feed_update failed: %w", err)
	}
	payload := update.Payload
	if len(payload) < 8 {
		return fmt.Errorf("unexpected feed payload length %d", len(payload))
	}
	timestamp := binary.BigEndian.Uint64(payload[:8])
	data := payload[8:]

	fmt.Printf("  index:      %d\n", update.Index)
	fmt.Printf("  index_next: %d\n", update.IndexNext)
	fmt.Printf("  timestamp:  %d (unix seconds)\n", timestamp)
	if utf8.Valid(data) {
		fmt.Printf("  payload:    %q\n", string(data))
	} else {
		fmt.Printf("  payload:    %d bytes (non-utf8)\n", len(data))
	}
	return nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
